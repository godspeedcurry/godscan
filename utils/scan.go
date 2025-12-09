// credit vscan
package utils

import (
	_ "embed"
	"encoding/hex"
	"errors"
	"fmt"
	"log"
	"net"
	"path/filepath"
	"regexp"
	"runtime"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/cheggaaa/pb/v3"
	regexp2 "github.com/dlclark/regexp2"

	"github.com/spf13/viper"
)

var config Config

var (
	inTargetChan  chan Target
	outResultChan chan Result
)

// 待探测的目标端口
type Target struct {
	IP       string `json:"ip"`
	Port     int    `json:"port"`
	Protocol string `json:"protocol"`
}

func (t *Target) GetAddress() string {
	return t.IP + ":" + strconv.Itoa(t.Port)
}

// 输出的结果数据
type Result struct {
	Target
	Service `json:"service"`

	Timestamp int32  `json:"timestamp"`
	Error     string `json:"error"`
}

// 获取的端口服务信息
type Service struct {
	Target

	Name        string `json:"name"`
	Protocol    string `json:"protocol"`
	Banner      string `json:"banner"`
	BannerBytes []byte `json:"banner_bytes"`

	//IsSSL	    bool `json:"is_ssl"`

	Extras  `json:"extras"`
	Details `json:"details"`
}

// 对应 NMap versioninfo 信息
type Extras struct {
	VendorProduct   string `json:"vendor_product,omitempty"`
	Version         string `json:"version,omitempty"`
	Info            string `json:"info,omitempty"`
	Hostname        string `json:"hostname,omitempty"`
	OperatingSystem string `json:"operating_system,omitempty"`
	DeviceType      string `json:"device_type,omitempty"`
	CPE             string `json:"cpe,omitempty"`
}

// 详细的结果数据（包含具体的 Probe 和匹配规则信息）
type Details struct {
	ProbeName     string `json:"probe_name"`
	ProbeData     string `json:"probe_data"`
	MatchMatched  string `json:"match_matched"`
	IsSoftMatched bool   `json:"soft_matched"`
}

// nmap-service-probes 中每一条规则
type Match struct {
	IsSoft bool

	Service     string
	Pattern     string
	VersionInfo string

	PatternCompiled *regexp2.Regexp
}

// 对获取到的 Banner 进行匹配
func (m *Match) MatchPattern(response []byte) bool {
	responseStr := string([]rune(string(response)))
	foundItems, err := m.PatternCompiled.FindStringMatch(responseStr)
	if err != nil {
		return false
	}
	// 匹配结果大于 0 表示规则与 response 匹配成功
	if foundItems != nil && len(foundItems.String()) > 0 {
		return true
	}
	return false
}

func (m *Match) ParseVersionInfo(response []byte) Extras {
	extras := Extras{}
	v := m.fillVersionInfoPlaceholders(response)
	extras.VendorProduct = extractSingle(v, `p/([^/]*)/`, `p|([^|]*)|`)
	extras.Version = extractSingle(v, `v/([^/]*)/`, `v|([^|]*)|`)
	extras.Info = extractSingle(v, `i/([^/]*)/`, `i|([^|]*)|`)
	extras.Hostname = extractSingle(v, `h/([^/]*)/`, `h|([^|]*)|`)
	extras.OperatingSystem = extractSingle(v, `o/([^/]*)/`, `o|([^|]*)|`)
	extras.DeviceType = extractSingle(v, `d/([^/]*)/`, `d|([^|]*)|`)
	extras.CPE = extractCPE(v)
	return extras
}

func (m *Match) fillVersionInfoPlaceholders(response []byte) string {
	responseStr := string([]rune(string(response)))
	foundItems, err := m.PatternCompiled.FindStringMatch(responseStr)
	if err != nil {
		Error("%s", err)
	}
	versionInfo := m.VersionInfo
	if foundItems == nil {
		return versionInfo
	}
	foundItemsList := foundItems.Groups()[1:]
	for index, value := range foundItemsList {
		dollarName := "$" + strconv.Itoa(index+1)
		versionInfo = strings.Replace(versionInfo, dollarName, value.String(), -1)
	}
	return versionInfo
}

func extractSingle(source string, patterns ...string) string {
	for _, p := range patterns {
		regex := regexp.MustCompile(p)
		m := regex.FindStringSubmatch(source)
		if len(m) > 1 {
			return m[1]
		}
	}
	return ""
}

func extractCPE(source string) string {
	for _, p := range []string{`cpe:/([^/]*)/`, `cpe:|([^|]*)|`} {
		regex := regexp.MustCompile(p)
		cpe := regex.FindStringSubmatch(source)
		if len(cpe) > 1 {
			return cpe[1]
		}
		if len(cpe) > 0 {
			return cpe[0]
		}
	}
	return ""
}

// 探针规则，包含该探针规则下的服务匹配条目和其他探测信息
type Probe struct {
	Name        string
	Data        string
	DecodedData []byte
	Protocol    string

	Ports    string
	SSLPorts string

	TotalWaitMS  int
	TCPWrappedMS int
	Rarity       int
	Fallback     string

	Matchs *[]Match
}

func isHexCode(b []byte) bool {
	matchRe := regexp.MustCompile(`\\x[0-9a-fA-F]{2}`)
	return matchRe.Match(b)
}

func isOctalCode(b []byte) bool {
	matchRe := regexp.MustCompile(`\\[0-7]{1,3}`)
	return matchRe.Match(b)
}

func isStructCode(b []byte) bool {
	matchRe := regexp.MustCompile(`\\[aftnrv]`)
	return matchRe.Match(b)
}

func isReChar(n int64) bool {
	reChars := `.*?+{}()^$|\[]`
	for _, char := range reChars {
		if n == int64(char) {
			return true
		}
	}
	return false
}

func isOtherEscapeCode(b []byte) bool {
	matchRe := regexp.MustCompile(`\\[^\\]`)
	return matchRe.Match(b)
}

/*
解析 nmap-service-probes 中匹配规则字符串，转换成 golang 中可以进行编译的字符串

	  e.g.
		(1) pattern: \0\xffHi
			decoded: []byte{0, 255, 72, 105} 4len

		(2) pattern: \\0\\xffHI
			decoded: []byte{92, 0, 92, 120, 102, 102, 72, 105} 8len

		(3) pattern: \x2e\x2a\x3f\x2b\x7b\x7d\x28\x29\x5e\x24\x7c\x5c
			decodedStr: \.\*\?\+\{\}\(\)\^\$\|\\
*/
func GetStructCodeMap() map[int][]byte {
	return map[int][]byte{
		97:  {0x07}, // \a
		102: {0x0c}, // \f
		116: {0x09}, // \t
		110: {0x0a}, // \n
		114: {0x0d}, // \r
		118: {0x0b}, // \v
	}
}
func DecodePattern(s string) ([]byte, error) {
	sByteOrigin := []byte(s)
	matchRe := regexp.MustCompile(`\\(x[0-9a-fA-F]{2}|[0-7]{1,3}|[aftnrv])`)
	sByteDec := matchRe.ReplaceAllFunc(sByteOrigin, func(match []byte) (v []byte) {
		var replace []byte
		// 十六进制转义格式
		if isHexCode(match) {
			hexNum := match[2:]
			byteNum, _ := strconv.ParseInt(string(hexNum), 16, 32)
			if isReChar(byteNum) {
				replace = []byte{'\\', uint8(byteNum)}
			} else {
				replace = []byte{uint8(byteNum)}
			}
		}
		// 格式控制符 \r\n\a\b\f\t
		if isStructCode(match) {
			structCodeMap := GetStructCodeMap()
			replace = structCodeMap[int(match[1])]
		}
		// 八进制转义格式
		if isOctalCode(match) {
			octalNum := match[2:]
			byteNum, _ := strconv.ParseInt(string(octalNum), 8, 32)
			replace = []byte{uint8(byteNum)}
		}
		return replace
	})

	matchRe2 := regexp.MustCompile(`\\([^\\])`)
	sByteDec2 := matchRe2.ReplaceAllFunc(sByteDec, func(match []byte) (v []byte) {
		var replace []byte
		if isOtherEscapeCode(match) {
			replace = match
		} else {
			replace = match
		}
		return replace
	})
	return sByteDec2, nil
}

func DecodeData(s string) ([]byte, error) {
	sByteOrigin := []byte(s)
	matchRe := regexp.MustCompile(`\\(x[0-9a-fA-F]{2}|[0-7]{1,3}|[aftnrv])`)
	sByteDec := matchRe.ReplaceAllFunc(sByteOrigin, func(match []byte) (v []byte) {
		var replace []byte
		// 十六进制转义格式
		if isHexCode(match) {
			hexNum := match[2:]
			byteNum, _ := strconv.ParseInt(string(hexNum), 16, 32)
			replace = []byte{uint8(byteNum)}
		}
		// 格式控制符 \r\n\a\b\f\t
		if isStructCode(match) {
			structCodeMap := GetStructCodeMap()
			replace = structCodeMap[int(match[1])]
		}
		// 八进制转义格式
		if isOctalCode(match) {
			octalNum := match[2:]
			byteNum, _ := strconv.ParseInt(string(octalNum), 8, 32)
			replace = []byte{uint8(byteNum)}
		}
		return replace
	})

	matchRe2 := regexp.MustCompile(`\\([^\\])`)
	sByteDec2 := matchRe2.ReplaceAllFunc(sByteDec, func(match []byte) (v []byte) {
		var replace []byte
		if isOtherEscapeCode(match) {
			replace = match
		} else {
			replace = match
		}
		return replace
	})
	return sByteDec2, nil
}

type Directive struct {
	DirectiveName string
	Flag          string
	Delimiter     string
	DirectiveStr  string
}

func (p *Probe) getDirectiveSyntax(data string) (directive Directive, err error) {
	directive = Directive{}

	if strings.Count(data, " ") <= 0 {
		return directive, fmt.Errorf("directive format error")
	}
	blankIndex := strings.Index(data, " ")
	if blankIndex < 0 || blankIndex+3 > len(data) {
		return directive, fmt.Errorf("directive split error")
	}
	directiveName := data[:blankIndex]
	Flag := data[blankIndex+1 : blankIndex+2]
	delimiter := data[blankIndex+2 : blankIndex+3]
	directiveStr := data[blankIndex+3:]

	directive.DirectiveName = directiveName
	directive.Flag = Flag
	directive.Delimiter = delimiter
	directive.DirectiveStr = directiveStr

	return directive, nil
}

func (p *Probe) getMatch(data string) (match Match, err error) {
	match = Match{}

	matchText := data[len("match")+1:]
	directive, derr := p.getDirectiveSyntax(matchText)
	if derr != nil {
		return match, derr
	}

	textSplited := strings.Split(directive.DirectiveStr, directive.Delimiter)

	pattern, versionInfo := textSplited[0], strings.Join(textSplited[1:], "")

	patternUnescaped, _ := DecodePattern(pattern)
	patternUnescapedStr := string([]rune(string(patternUnescaped)))
	patternCompiled, ok := regexp2.Compile(patternUnescapedStr, regexp2.None)
	if ok != nil {
		Error("Parse match data failed, data: %s %s", data, patternUnescapedStr)
		return match, ok
	}

	match.Service = directive.DirectiveName
	match.Pattern = pattern
	match.PatternCompiled = patternCompiled
	match.VersionInfo = versionInfo

	return match, nil
}

func (p *Probe) getSoftMatch(data string) (softMatch Match, err error) {
	softMatch = Match{IsSoft: true}

	matchText := data[len("softmatch")+1:]
	directive, derr := p.getDirectiveSyntax(matchText)
	if derr != nil {
		return softMatch, derr
	}

	textSplited := strings.Split(directive.DirectiveStr, directive.Delimiter)

	pattern, versionInfo := textSplited[0], strings.Join(textSplited[1:], "")
	patternUnescaped, _ := DecodePattern(pattern)
	patternUnescapedStr := string([]rune(string(patternUnescaped)))
	patternCompiled, ok := regexp2.Compile(patternUnescapedStr, regexp2.None)
	if ok != nil {
		Error("Parse softmatch data failed, data: %s", data)
		return softMatch, ok
	}

	softMatch.Service = directive.DirectiveName
	softMatch.Pattern = pattern
	softMatch.PatternCompiled = patternCompiled
	softMatch.VersionInfo = versionInfo

	return softMatch, nil
}

func (p *Probe) parsePorts(data string) {
	p.Ports = data[len("ports")+1:]
}

func (p *Probe) parseSSLPorts(data string) {
	p.SSLPorts = data[len("sslports")+1:]
}

func (p *Probe) parseTotalWaitMS(data string) {
	p.TotalWaitMS, _ = strconv.Atoi(string(data[len("totalwaitms")+1:]))
}

func (p *Probe) parseTCPWrappedMS(data string) {
	p.TCPWrappedMS, _ = strconv.Atoi(string(data[len("tcpwrappedms")+1:]))
}

func (p *Probe) parseRarity(data string) {
	p.Rarity, _ = strconv.Atoi(string(data[len("rarity")+1:]))
}

func (p *Probe) parseFallback(data string) {
	p.Fallback = data[len("fallback")+1:]
}

func (p *Probe) fromString(data string) error {
	var err error

	data = strings.TrimSpace(data)
	lines := strings.Split(data, "\n")
	probeStr := lines[0]

	if err = p.parseProbeInfo(probeStr); err != nil {
		return err
	}

	var matchs []Match
	for _, line := range lines {
		if strings.HasPrefix(line, "match ") {
			match, err := p.getMatch(line)
			if err != nil {
				continue
			}
			matchs = append(matchs, match)
		} else if strings.HasPrefix(line, "softmatch ") {
			softMatch, err := p.getSoftMatch(line)
			if err != nil {
				continue
			}
			matchs = append(matchs, softMatch)
		} else if strings.HasPrefix(line, "ports ") {
			//p.Ports = getPorts(line)
			p.parsePorts(line)
		} else if strings.HasPrefix(line, "sslports ") {
			//p.SSLPorts = getSSLPorts(line)
			p.parseSSLPorts(line)
		} else if strings.HasPrefix(line, "totalwaitms ") {
			//p.TotalWaitMS = getTotalWaitMS(line)
			p.parseTotalWaitMS(line)
		} else if strings.HasPrefix(line, "tcpwrappedms ") {
			//p.TCPWrappedMS = getTCPWrappedMS(line)
			p.parseTCPWrappedMS(line)
		} else if strings.HasPrefix(line, "rarity ") {
			//p.Rarity = getRarity(line)
			p.parseRarity(line)
		} else if strings.HasPrefix(line, "fallback ") {
			//p.Fallback = getFallback(line)
			p.parseFallback(line)
		}
	}
	p.Matchs = &matchs
	return err
}

func (p *Probe) parseProbeInfo(probeStr string) error {
	proto := probeStr[:4]
	other := probeStr[4:]

	if !(proto == "TCP " || proto == "UDP ") {
		return fmt.Errorf("unsupported protocol")
	}
	if len(other) == 0 {
		return fmt.Errorf("bad probe name")
	}

	directive, derr := p.getDirectiveSyntax(other)
	if derr != nil {
		return derr
	}

	p.Name = directive.DirectiveName
	p.Data = strings.Split(directive.DirectiveStr, directive.Delimiter)[0]
	p.DecodedData, _ = DecodeData(p.Data)
	p.Protocol = strings.ToLower(strings.TrimSpace(proto))
	return nil
}

func (p *Probe) ContainsPort(testPort int) bool {
	ports := strings.Split(p.Ports, ",")

	// 常规分割判断，Ports 字符串不含端口范围形式 "[start]-[end]"
	for _, port := range ports {
		cmpPort, _ := strconv.Atoi(port)
		if testPort == cmpPort {
			return true
		}
	}
	// 范围判断检查，拆分 Ports 中诸如 "[start]-[end]" 类型的端口范围进行比较
	for _, port := range ports {
		if strings.Contains(port, "-") {
			portRange := strings.Split(port, "-")
			start, _ := strconv.Atoi(portRange[0])
			end, _ := strconv.Atoi(portRange[1])
			for cmpPort := start; cmpPort <= end; cmpPort++ {
				if testPort == cmpPort {
					return true
				}
			}
		}
	}
	return false
}

func (p *Probe) ContainsSSLPort(testPort int) bool {
	ports := strings.Split(p.SSLPorts, ",")

	// 常规分割判断，Ports 字符串不含端口范围形式 "[start]-[end]"
	for _, port := range ports {
		cmpPort, _ := strconv.Atoi(port)
		if testPort == cmpPort {
			return true
		}
	}
	// 范围判断检查，拆分 Ports 中诸如 "[start]-[end]" 类型的端口范围进行比较
	for _, port := range ports {
		if strings.Contains(port, "-") {
			portRange := strings.Split(port, "-")
			start, _ := strconv.Atoi(portRange[0])
			end, _ := strconv.Atoi(portRange[1])
			for cmpPort := start; cmpPort <= end; cmpPort++ {
				if testPort == cmpPort {
					return true
				}
			}
		}
	}
	return false
}

// ProbesRarity 用于使用 sort 对 Probe 对象按 Rarity 属性值进行排序
type ProbesRarity []Probe

func (ps ProbesRarity) Len() int {
	return len(ps)
}

func (ps ProbesRarity) Swap(i, j int) {
	ps[i], ps[j] = ps[j], ps[i]
}

func (ps ProbesRarity) Less(i, j int) bool {
	return ps[i].Rarity < ps[j].Rarity
}

func sortProbesByRarity(probes []Probe) (probesSorted []Probe) {
	probesToSort := ProbesRarity(probes)
	sort.Stable(probesToSort)
	// 稳定排序 ， 探针发送顺序不同，最后会导致探测服务出现问题
	probesSorted = []Probe(probesToSort)
	return probesSorted
}

type VScan struct {
	Exclude string

	Probes []Probe

	ProbesMapKName map[string]Probe
}

func (v *VScan) parseProbesFromContent(content string) {
	var probes []Probe

	var lines []string
	// 过滤掉规则文件中的注释和空行
	linesTemp := strings.Split(content, "\n")
	for _, lineTemp := range linesTemp {
		lineTemp = strings.TrimSpace(lineTemp)
		if lineTemp == "" || strings.HasPrefix(lineTemp, "#") {
			continue
		}
		lines = append(lines, lineTemp)
	}
	// 判断第一行是否为 "Exclude " 设置
	if len(lines) == 0 {
		Warning("nmap-service-probes content empty")
		v.Probes = probes
		return
	}
	c := 0
	for _, line := range lines {
		if strings.HasPrefix(line, "Exclude ") {
			c += 1
		}
		// 一份规则文件中有且至多有一个 Exclude 设置
		if c > 1 {
			Warning("multiple Exclude directives, using the first one")
		}
	}
	l := lines[0]
	if !(strings.HasPrefix(l, "Exclude ") || strings.HasPrefix(l, "Probe ")) {
		Warning("invalid probe file header")
		v.Probes = probes
		return
	}
	if c == 1 {
		v.Exclude = l[len("Exclude")+1:]
		lines = lines[1:]
	}
	content = strings.Join(lines, "\n")
	content = "\n" + content

	// 按 "\nProbe" 拆分探针组内容
	probeParts := strings.Split(content, "\nProbe")
	probeParts = probeParts[1:]

	for _, probePart := range probeParts {
		probe := Probe{}
		err := probe.fromString(probePart)
		if err != nil {
			log.Println(err)
			continue
		}
		probes = append(probes, probe)

	}
	v.Probes = probes
}

func (v *VScan) parseProbesToMapKName() {
	var probesMap = map[string]Probe{}
	for _, probe := range v.Probes {
		probesMap[probe.Name] = probe
	}
	v.ProbesMapKName = probesMap
}

//go:embed custom-probes
var customProbes string

//go:embed nmap-service-probes
var nmapProbes string

// 从文件中解析并加载 Probes 初始化 VScan 实例
func (v *VScan) Init() {
	// 读取 nmap-service-probes 和 自定义规则文件
	// 解析规则文本得到 Probe 列表
	v.parseProbesFromContent(nmapProbes + "\n" + customProbes)
	// 按 Probe Name 建立 Map 方便后续 Fallback 快速访问
	v.parseProbesToMapKName()
}

// VScan 探测时的参数配置
type Config struct {
	Rarity   int
	Routines int

	SendTimeout time.Duration
	ReadTimeout time.Duration

	NULLProbeOnly bool
	UseAllProbes  bool
	SSLAlwaysTry  bool
}

// VScan 探测目标端口函数，返回探测结果和错误信息
// 1. probes ports contains port
// 2. probes sslports contains port
// 3. probes ports contains port use ssl try to
func (v *VScan) Explore(target Target, config *Config) (Result, error) {
	var probesUsed []Probe
	// 使用所有 Probe 探针进行服务识别尝试，忽略 Probe 的 Ports 端口匹配
	if config.UseAllProbes {
		for _, probe := range v.Probes {
			if strings.EqualFold(probe.Protocol, target.Protocol) {
				probesUsed = append(probesUsed, probe)
			}
		}
		//probesUsed = v.Probes
	} else
	// 配置仅使用 NULL Probe 进行探测，及不发送任何 Data，只监听端口返回数据
	if config.NULLProbeOnly {
		probesUsed = append(probesUsed, v.ProbesMapKName["NULL"])
	} else
	// 未进行特殊配置，默认只使用 NULL Probe 和包含了探测端口的 Probe 探针组
	{
		for _, probe := range v.Probes {
			if probe.ContainsPort(target.Port) && strings.EqualFold(probe.Protocol, target.Protocol) {
				probesUsed = append(probesUsed, probe)
			}
		}
		// 将默认 NULL Probe 添加到探针列表
		probesUsed = append(probesUsed, v.ProbesMapKName["NULL"])
		re := regexp.MustCompile(`Probe TCP (\w+)`)
		matches := re.FindAllStringSubmatch(customProbes, -1)

		// 打印匹配项
		for _, match := range matches {
			probesUsed = append(probesUsed, v.ProbesMapKName[match[1]])
		}

	}

	// 按 Probe 的 Rarity 升序排列
	probesUsed = sortProbesByRarity(probesUsed)

	// 根据 Config 配置舍弃 probe.Rarity > config.Rarity 的探针
	var probesUsedFiltered []Probe
	for _, probe := range probesUsed {
		if probe.Rarity > config.Rarity {
			continue
		}
		probesUsedFiltered = append(probesUsedFiltered, probe)
	}
	probesUsed = probesUsedFiltered

	result, err := v.scanWithProbes(target, &probesUsed, config)

	return result, err
}

func (v *VScan) scanWithProbes(target Target, probes *[]Probe, config *Config) (Result, error) {
	var result = Result{Target: target}
	for _, probe := range *probes {
		Debug("Try Probe(%s), Data(%s)", probe.Name, probe.Data)
		response, _ := grabResponse(target, probe.DecodedData, config)
		if len(response) == 0 {
			continue
		}
		Debug("Get response %d bytes from destination with Probe(%s)", len(response), probe.Name)
		res, matched := v.matchProbe(target, probe, response)
		if matched {
			return res, nil
		}
	}
	return result, errEmptyResponse
}

func (v *VScan) matchProbe(target Target, probe Probe, response []byte) (Result, bool) {
	if probe.Matchs != nil {
		if res, ok := applyMatches(target, probe, response, probe.Matchs, false); ok {
			return res, true
		}
	}

	if res, ok := v.tryFallback(target, probe, response); ok {
		return res, true
	}

	if res, ok := buildUnknown(target, probe, response); ok {
		return res, true
	}
	return Result{}, false
}

func applyMatches(target Target, probe Probe, response []byte, matches *[]Match, markSoft bool) (Result, bool) {
	var soft *Match
	for _, match := range *matches {
		matched := match.MatchPattern(response)
		if matched && !match.IsSoft {
			return buildMatchResult(target, probe, match, response, false), true
		}
		if matched && match.IsSoft && soft == nil {
			Info("Soft matched: %s, pattern: %s", match.Service, match.Pattern)
			soft = &match
		}
	}
	if soft != nil {
		return buildMatchResult(target, probe, *soft, response, true), true
	}
	return Result{}, false
}

func (v *VScan) tryFallback(target Target, probe Probe, response []byte) (Result, bool) {
	fallback := probe.Fallback
	fbProbe, ok := v.ProbesMapKName[fallback]
	if !ok || fbProbe.Matchs == nil {
		return Result{}, false
	}
	if res, ok := applyMatches(target, probe, response, fbProbe.Matchs, true); ok {
		return res, true
	}
	return Result{}, false
}

func buildUnknown(target Target, probe Probe, response []byte) (Result, bool) {
	result := Result{Target: target}
	result.Service.Target = target
	result.Service.Protocol = strings.ToLower(probe.Protocol)
	result.Service.Details.ProbeName = probe.Name
	result.Service.Details.ProbeData = probe.Data
	result.Banner = string(response)
	result.BannerBytes = response
	result.Service.Name = "unknown"
	result.Timestamp = int32(time.Now().Unix())
	return result, true
}

func buildMatchResult(target Target, probe Probe, match Match, response []byte, soft bool) Result {
	extras := match.ParseVersionInfo(response)
	result := Result{Target: target}
	result.Service.Target = target
	result.Service.Details.ProbeName = probe.Name
	result.Service.Details.ProbeData = probe.Data
	result.Service.Details.MatchMatched = match.Pattern
	result.Service.Details.IsSoftMatched = soft

	result.Service.Protocol = strings.ToLower(probe.Protocol)
	result.Service.Name = match.Service

	result.Banner = string(response)
	result.BannerBytes = response
	result.Service.Extras = extras
	result.Timestamp = int32(time.Now().Unix())
	return result
}

func grabResponse(target Target, data []byte, config *Config) ([]byte, error) {
	var response []byte

	addr := target.GetAddress()
	dialer := net.Dialer{}

	proto := target.Protocol
	if !(proto == "tcp" || proto == "udp") {
		log.Fatal("Failed to send request with unknown protocol", proto)
	}

	conn, errConn := dialer.Dial(proto, addr)
	if errConn != nil {
		return response, errConn
	}
	defer conn.Close()

	if len(data) > 0 {
		conn.SetWriteDeadline(time.Now().Add(config.SendTimeout))
		_, errWrite := conn.Write(data)
		if errWrite != nil {
			return response, errWrite
		}
	}

	conn.SetReadDeadline(time.Now().Add(config.ReadTimeout))
	for {
		buff := make([]byte, 1024)
		n, errRead := conn.Read(buff)
		if errRead != nil {
			if len(response) > 0 {
				break
			} else {
				return response, errRead
			}
		}
		if n > 0 {
			response = append(response, buff[:n]...)
		}
	}

	return response, nil
}

// 错误类型
var (
	errEmptyResponse = errors.New("empty response fetched from destination'")
)

func ConfigInit() {

	config.Routines = 10

	config.Rarity = viper.GetInt("scan-rarity")

	config.SendTimeout = time.Duration(viper.GetInt("scan-send-timeout")) * time.Second
	config.ReadTimeout = time.Duration(viper.GetInt("scan-read-timeout")) * time.Second

	config.UseAllProbes = viper.GetBool("all-probe")
	config.NULLProbeOnly = viper.GetBool("null-probe-only")

}

type Worker struct {
	In     chan Target
	Out    chan Result
	Config *Config
}

func (w *Worker) Start(v *VScan, wg *sync.WaitGroup) {
	go func() {
		for {
			target, ok := <-w.In
			if !ok {
				break
			}
			result, err := v.Explore(target, w.Config)
			if err != nil {
				continue
			}
			if err == errEmptyResponse {
				continue
			}
			w.Out <- result
		}
		wg.Done()
	}()
}

func ScanWithIpAndPort(addr []ProtocolInfo) {
	Info("Total addr(s): %d", len(addr))
	ConfigInit()
	runtime.GOMAXPROCS(runtime.NumCPU())
	bar := pb.StartNew(len(addr))
	// 初始化 VScan 实例，并加载默认 nmap-service-probes 文件解析 Probe 列表
	v := VScan{}
	v.Init()

	// 输入输出缓冲为最大协程数量的 5 倍
	inTargetChan = make(chan Target, config.Routines*5)
	outResultChan = make(chan Result, config.Routines*2)

	// 最大协程并发量为参数 config.Routines
	wgWorkers := sync.WaitGroup{}
	wgWorkers.Add(int(config.Routines))

	// 启动协程并开始监听处理输入的 Target
	for i := 0; i < config.Routines; i++ {
		worker := Worker{inTargetChan, outResultChan, &config}
		worker.Start(&v, &wgWorkers)
	}
	ServiceInfoResults := []string{}
	// 实时结果输出协程
	wgOutput := sync.WaitGroup{}
	wgOutput.Add(1)
	var mu sync.Mutex

	go func(wg *sync.WaitGroup) {
		for {
			result, ok := <-outResultChan
			if ok {
				// 对获取到的 Result 进行判断，如果含有 Error 信息则进行筛选输出
				banner := result.Banner
				if len(banner) > 128 {
					banner = banner[:128]
				}
				bar.Increment()
				ServiceInfoResult := fmt.Sprintf("%s://%s:%d", result.Name, result.Target.IP, result.Target.Port)
				mu.Lock()
				ServiceInfoResults = append(ServiceInfoResults, ServiceInfoResult)
				mu.Unlock()
				Success("%s", ServiceInfoResult)
				Warning("%s", hex.Dump([]byte(banner)))
			} else {
				break
			}
		}
		wg.Done()
	}(&wgOutput)

	for _, a := range addr {
		target := Target{
			IP:       a.Ip,
			Port:     a.Port,
			Protocol: "tcp",
		}
		inTargetChan <- target
	}
	close(inTargetChan)
	wgWorkers.Wait()
	Debug("All workers exited")
	close(outResultChan)
	Debug("Output goroutine finished")
	wgOutput.Wait()

	sort.Strings(ServiceInfoResults)
	outDir := viper.GetString("output-dir")
	if outDir == "" {
		outDir = "."
	}
	servicePath := filepath.Join(outDir, "service.txt")
	Success("Log at %s", servicePath)
	FileWrite(servicePath, "%s", strings.Join(ServiceInfoResults, "\n")+"\n")

}
