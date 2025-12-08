package utils

import (
	"bytes"
	"crypto/md5"
	"database/sql"
	_ "embed"
	"errors"
	"fmt"
	"hash"
	"io"
	"net"
	"os"
	"path"
	"sync"
	"time"

	"github.com/godspeedcurry/godscan/common"
	prettytable "github.com/jedib0t/go-pretty/v6/table"
	"github.com/spf13/viper"

	"net/http"
	"net/url"
	"regexp"
	"strings"

	b64 "encoding/base64"
	"encoding/hex"
	"encoding/json"
	"sort"
	"strconv"

	"github.com/PuerkitoBio/goquery"

	mapset "github.com/deckarep/golang-set"
	"github.com/fatih/color"
	"github.com/twmb/murmur3"
)

var sensitiveUrl sync.Map
var importantApiSeen sync.Map

type SpiderSummary struct {
	URL         string
	Title       string
	Finger      string
	ContentType string
	Status      int
	Length      int
	Keyword     string
	SimHash     string
	IconHash    string
	ApiCount    int
	UrlCount    int
	CDNCount    int
	CDNHosts    string
	SaveDir     string
	Err         error
}

func Mmh3Hash32(raw []byte) string {
	var h32 hash.Hash32 = murmur3.New32()
	_, err := h32.Write([]byte(raw))
	if err == nil {
		return fmt.Sprintf("%d", int32(h32.Sum32()))
	} else {
		return "0"
	}
}
func HttpGetServerHeader(Url string, NeedTitle bool, Method string) (string, string, string, error) {
	req, err := http.NewRequest(Method, Url, nil)
	if err != nil {
		return "", "", "", err
	}
	if Method == http.MethodPost {
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	}
	SetHeaders(req)
	resp, err := Client.Do(req)
	if err != nil {
		logRequestError(Url, err)
		return "", "", "", err
	}
	defer resp.Body.Close()
	doc, err := goquery.NewDocumentFromReader(resp.Body)
	if err != nil {
		logRequestError(Url, err)
		return "", "", "", err
	}

	title := doc.Find("title").Text()
	ServerValue := resp.Header["Server"]
	Status := resp.Status
	retServerValue := ""
	if len(ServerValue) != 0 {
		retServerValue = ServerValue[0]
	}
	return retServerValue, Status, title, nil
}

func IsVuePath(Path string) bool {
	// reg := regexp.MustCompile(`(app|index|config|main|chunk)`)
	// res := reg.FindAllString(Path, -1)
	return strings.HasSuffix(Path, ".js") && !strings.HasSuffix(Path, ".min.js")
}

func HighLight(data string, keywords []string, fingers []string, Url string) {
	var output bool = false
	for _, keyword := range keywords {
		re := regexp.MustCompile("(?i)" + Quote(keyword))
		if len(re.FindAllString(data, -1)) > 0 {
			output = true
			data = re.ReplaceAllString(data, color.RedString(keyword))
		}
	}
	for _, keyword := range fingers {
		re := regexp.MustCompile("(?i)(" + Quote(keyword) + ")")
		if len(re.FindAllString(data, -1)) > 0 {
			output = true
			data = re.ReplaceAllString(data, color.RedString("$1"))
		}
	}
	if output {
		Info("%s\n%s", Url, data)
	}
}

func uselessUrl(Url string, Depth int) bool {
	if Depth == 0 || Url == "" || !isValidUrl(Url) {
		return true
	}
	return false
}

func isJSAssetPath(p string) bool {
	lp := strings.ToLower(p)
	return strings.HasSuffix(lp, ".js") || strings.HasSuffix(lp, ".mjs")
}

func buildSourceMapURL(jsURL string) string {
	u, err := url.Parse(jsURL)
	if err != nil || u.Path == "" {
		return ""
	}
	lp := strings.ToLower(u.Path)
	if strings.HasSuffix(lp, ".map") || !isJSAssetPath(u.Path) {
		return ""
	}
	u.Path = u.Path + ".map"
	return u.String()
}

func fetchSourceMap(mapURL string) (int, int, error) {
	// Prefer HEAD to avoid large transfers; fall back to GET if HEAD fails hard.
	doReq := func(method string) (int, int, error) {
		req, err := http.NewRequest(method, mapURL, nil)
		if err != nil {
			return -1, 0, err
		}
		SetHeaders(req)
		resp, err := Client.Do(req)
		if err != nil {
			return -1, 0, err
		}
		defer resp.Body.Close()
		maxBody := viper.GetInt("max-body-bytes")
		if maxBody <= 0 || maxBody > 4*1024*1024 {
			maxBody = 4 * 1024 * 1024
		}
		body, _ := io.ReadAll(io.LimitReader(resp.Body, int64(maxBody)))
		length := len(body)
		if cl := resp.Header.Get("Content-Length"); cl != "" {
			if v, e := strconv.Atoi(cl); e == nil {
				length = v
			}
		}
		return resp.StatusCode, length, nil
	}

	status, length, err := doReq(http.MethodHead)
	if err == nil && status > 0 {
		return status, length, nil
	}

	return doReq(http.MethodGet)
}

func sameHost(rootURL, target string) bool {
	root, err := url.Parse(rootURL)
	if err != nil {
		return true
	}
	t, err := url.Parse(target)
	if err != nil {
		return false
	}
	return strings.EqualFold(root.Hostname(), t.Hostname())
}

func sourceMapFromContent(jsURL string, body string) string {
	// Look for //# sourceMappingURL=... or //@ sourceMappingURL=...
	re := regexp.MustCompile(`(?m)sourceMappingURL=([^\s]+)`)
	m := re.FindStringSubmatch(body)
	if len(m) < 2 {
		return ""
	}
	raw := strings.TrimSpace(m[1])
	if raw == "" {
		return ""
	}
	u, err := url.Parse(raw)
	if err != nil {
		return ""
	}
	if u.IsAbs() {
		return u.String()
	}
	base, err := url.Parse(jsURL)
	if err != nil {
		return ""
	}
	return base.ResolveReference(u).String()
}

func probeSourceMap(rootURL, jsURL, saveDir string, seen mapset.Set, db *sql.DB) {
	mapURL := buildSourceMapURL(jsURL)
	if mapURL == "" {
		return
	}
	if seen != nil && seen.Contains(mapURL) {
		return
	}
	if seen != nil {
		seen.Add(mapURL)
	}
	if !sameHost(rootURL, mapURL) {
		return
	}
	status, length, err := fetchSourceMap(mapURL)
	if err != nil || status < 0 || status >= 400 {
		return
	}
	Info("source map found: %s (status=%d size=%d) from %s", mapURL, status, length, jsURL)
	FileWrite(saveDir+"sourcemaps.txt", "%s\t%d\t%d\t%s\n", mapURL, status, length, jsURL)
	_ = SaveSourceMaps(db, []SourceMapHit{{
		RootURL: rootURL,
		JSURL:   jsURL,
		MapURL:  mapURL,
		Status:  status,
		Length:  length,
	}})
}

func parseDir(fullPath string, MaxDepth int) []string {
	// 去除末尾的斜杠, 最多两层
	dirs := []string{}
	// 逐级获取目录
	for {
		dir, _ := path.Split(fullPath)
		// 如果已经到达最顶层目录，则退出循环
		if dir == "/" {
			break
		}
		// 更新 fullPath 为父目录路径
		fullPath = strings.TrimSuffix(dir, "/")
		if strings.Count(dir, "/") <= (MaxDepth + 1) {
			dirs = append(dirs, dir)
		}
	}
	return dirs
}

func isValidUrl(Url string) bool {
	arr := []string{
		"alicdn.com", "163.com", "nginx.com", "qq.com", "amap.com", "cnzz.com", "github.com", "apache.org", "gitlab.com", "centos.org",
		"fonts.googleapis.com", "fonts.gstatic.com", "gstatic.com", "w3.org", "cloudflare.com", "cdnjs.cloudflare.com",
		"logout", "delete", "drop", "remove", "clear", "clean", "purge", "erase", "discard", "unregister", "revoke",
	}
	for _, key := range arr {
		if strings.Contains(strings.ToLower(Url), key) {
			return false
		}
	}
	suffix := []string{".min.js", ".png", ".jpeg", ".jpg", ".gif", ".bmp", ".vue", ".css", ".ico", ".svg"}
	for _, ign := range suffix {
		if strings.Contains(Url, ign) {
			return false
		}
	}

	// 解析URL
	parsedURL, err := url.Parse(Url)
	if err != nil {
		return false
	}
	// 检查URL的格式是否合法
	if !parsedURL.IsAbs() || parsedURL.Hostname() == "" || (parsedURL.Scheme != "http" && parsedURL.Scheme != "https") {
		return false
	}
	if !viper.GetBool("private-ip") {
		ip := net.ParseIP(parsedURL.Hostname())
		if ip != nil {
			return !ip.IsPrivate()
		}
	}
	return true
}

func ImportantApiJudge(ApiResult string, Url string) {
	for _, key := range common.ImportantApi {
		if strings.Contains(ApiResult, key) {
			mark := Url + "|" + key
			if _, ok := importantApiSeen.Load(mark); ok {
				continue
			}
			importantApiSeen.Store(mark, true)
			Success("Import Api found %s @ %s", key, Url)
			if key == "/api/blade-user" {
				Success("Might related to SpringBlade CVE-2021-44910")
				FileWrite("cve.log", "[%s] Might related to SpringBlade CVE-2021-44910", Url)
			}
		}
	}
}

func filterOutUrl(Url string) bool {
	filters := viper.GetStringSlice("filter")

	for _, filter := range filters {
		if strings.Contains(Url, filter) {
			return true
		}
	}
	return false
}

func parseVueUrl(Url string, RootPath string, doc string, directory string, apiCounter *int, db *sql.DB) {
	quote := "['\"`]"
	ApiReg := regexp.MustCompile(quote + `[\w\$\{\}]*(?P<path>/[\w/\-\|_=@\?\:.]+?)` + quote)

	ApiResultTuple := ApiReg.FindAllStringSubmatch(strings.ReplaceAll(doc, "\\", ""), -1)
	ApiResult := []string{}

	for _, tmp := range ApiResultTuple {
		ApiResult = append(ApiResult, viper.GetString("ApiPrefix")+tmp[1])
	}
	ApiResult = RemoveDuplicatesString(ApiResult)
	ApiResultLen := len(ApiResult)

	if ApiResultLen > 0 {
		*apiCounter += ApiResultLen
		var totalResult = strings.Join(ApiResult, "\n")
		ImportantApiJudge(totalResult, Url)
		FileWrite(directory+"api_raw.txt", "==== %s ====\n# total: %d\n%s\n", Url, ApiResultLen, totalResult)
		SaveAPIPaths(db, RootPath, Url, ApiResult, directory)
	}

	subdirs := []string{}
	subdirMatches := []string{}
	// 子目录 + 少量敏感目录
	for _, apiPath := range ApiResult {
		subdirMatches = append(subdirMatches, parseDir(apiPath, 2)...)
	}

	subdirMatches = RemoveDuplicatesString(subdirMatches)
	for _, match := range subdirMatches {
		normalizeUrl := Normalize(match, RootPath)
		if !isValidUrl(normalizeUrl) {
			continue
		}
		subdirs = append(subdirs, normalizeUrl)
	}

	subdirs = RemoveDuplicatesString(subdirs)
	if len(subdirs) > 0 {
		FileWrite(directory+"sub_directory.txt", "%s\n", strings.Join(subdirs, "\n"))
		var wg sync.WaitGroup
		rows := make(chan []string)
		file, err := os.OpenFile(directory+"sub_directory.txt", os.O_APPEND|os.O_WRONLY|os.O_CREATE, 0644)
		if err != nil {
			Error("%s", err)
			return
		}
		table := prettytable.NewWriter()
		table.SetOutputMirror(file)
		table.AppendHeader(prettytable.Row{"Url", "Title", "Finger", "Content-Type", "StatusCode", "Length"})
		table.SetStyle(prettytable.StyleRounded)

		cnt := 0

		for _, line := range subdirs {
			for _, dir := range []string{".git/config", "swagger-resources", "v2/api-docs", ""} {
				cnt += 1
				if cnt >= 50 {
					continue
				}
				wg.Add(1)
				go func(url string, dir string) {
					defer wg.Done()
					ret := DirBrute(url, dir)
					rows <- ret
				}(line, dir)
			}
		}
		go func() {
			wg.Wait()
			close(rows)
		}()
		for ret := range rows {
			AddDataToTable(table, ret)
		}
		if table.Length() >= 1 {
			table.Render()
		}
		if _, ok := sensitiveUrl.Load(Url); !ok {
			sensitiveUrl.Store(Url, true)
			SensitiveInfoCollect(db, Url, doc, directory)
		}
	}
}

type prefetchedPage struct {
	body        string
	contentType string
}

func Spider(RootPath string, Url string, depth int, directory string, myMap mapset.Set, sourceMapSeen mapset.Set, apiCounter *int, db *sql.DB, cached *prefetchedPage) error {
	if uselessUrl(Url, depth) || filterOutUrl(Url) {
		return nil
	}
	myMap.Add(Url)
	u, err := url.Parse(Url)
	if err != nil {
		Error("%s", Url)
		return err
	}
	if isJSAssetPath(u.Path) {
		resp, err := fetchGet(Url)
		if err != nil {
			return err
		}
		defer resp.Body.Close()
		return handleJSAsset(RootPath, Url, u.Path, resp, directory, sourceMapSeen, apiCounter, db)
	}
	if cached != nil {
		return handleHTMLContent(RootPath, Url, depth, directory, myMap, sourceMapSeen, apiCounter, db, cached.body)
	}
	resp, err := fetchGet(Url)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	bodyStr := readBodyString(resp)
	return handleHTMLContent(RootPath, Url, depth, directory, myMap, sourceMapSeen, apiCounter, db, bodyStr)
}

func fetchGet(Url string) (*http.Response, error) {
	req, err := http.NewRequest(http.MethodGet, Url, nil)
	if err != nil {
		return nil, err
	}
	SetHeaders(req)
	return Client.Do(req)
}

func handleJSAsset(rootPath, fullURL, path string, resp *http.Response, directory string, sourceMapSeen mapset.Set, apiCounter *int, db *sql.DB) error {
	probeSourceMap(rootPath, fullURL, directory, sourceMapSeen, db)
	bodyStr := readBodyString(resp)
	if sm := sourceMapFromContent(fullURL, bodyStr); sm != "" {
		probeSourceMap(rootPath, sm, directory, sourceMapSeen, db)
	}
	if IsVuePath(path) {
		parseVueUrl(fullURL, rootPath, bodyStr, directory, apiCounter, db)
	}
	return nil
}

func readBodyString(resp *http.Response) string {
	maxBody := viper.GetInt("max-body-bytes")
	if maxBody <= 0 || maxBody > 4*1024*1024 {
		maxBody = 4 * 1024 * 1024
	}
	limited := io.LimitReader(resp.Body, int64(maxBody))
	bodyBuf, _ := io.ReadAll(limited)
	return string(bodyBuf)
}

func crawlLinks(doc *goquery.Document, rootPath string, depth int, directory string, myMap mapset.Set, sourceMapSeen mapset.Set, apiCounter *int, db *sql.DB) {
	doc.Find("a, link").Each(func(i int, selector *goquery.Selection) {
		href, _ := selector.Attr("href")
		if href == "" {
			return
		}
		normalizeUrl := Normalize(href, rootPath)
		if normalizeUrl != "" && !myMap.Contains(normalizeUrl) {
			Spider(rootPath, normalizeUrl, depth-1, directory, myMap, sourceMapSeen, apiCounter, db, nil)
		}
	})
	doc.Find("script, iframe").Each(func(i int, selector *goquery.Selection) {
		src, _ := selector.Attr("src")
		if src == "" {
			return
		}
		normalizeUrl := Normalize(src, rootPath)
		if goquery.NodeName(selector) == "script" {
			probeSourceMap(rootPath, normalizeUrl, directory, sourceMapSeen, db)
		}
		if normalizeUrl != "" && !myMap.Contains(normalizeUrl) {
			Spider(rootPath, normalizeUrl, depth-1, directory, myMap, sourceMapSeen, apiCounter, db, nil)
		}
	})
}

func handleHTMLContent(rootPath, Url string, depth int, directory string, myMap mapset.Set, sourceMapSeen mapset.Set, apiCounter *int, db *sql.DB, bodyStr string) error {
	doc, err := goquery.NewDocumentFromReader(strings.NewReader(bodyStr))
	if err != nil {
		Error("%s", err)
		return err
	}
	// 尝试从脚本中收集 SourceMap 线索
	doc.Find("script").Each(func(i int, selector *goquery.Selection) {
		if src, ok := selector.Attr("src"); ok {
			normalizeUrl := Normalize(src, rootPath)
			probeSourceMap(rootPath, normalizeUrl, directory, sourceMapSeen, db)
		}
	})
	if _, ok := sensitiveUrl.Load(Url); !ok {
		sensitiveUrl.Store(Url, true)
		SensitiveInfoCollect(db, Url, bodyStr, directory)
	}
	crawlLinks(doc, rootPath, depth, directory, myMap, sourceMapSeen, apiCounter, db)
	return nil
}

func DisplayHeader(Url string, Method string) {
	ServerHeader, Status, Title, err := HttpGetServerHeader(Url, true, Method)
	if err != nil {
		Error("Error: %s", err)
	} else {
		Info("url=\"%s\" method=\"%s\" server=\"%s\" status=\"%s\" title=\"%s\"", Url, Method, ServerHeader, Status, Title)
	}
}

func StandBase64(braw []byte) []byte {
	bckd := b64.StdEncoding.EncodeToString(braw)
	var buffer bytes.Buffer
	for i := 0; i < len(bckd); i++ {
		ch := bckd[i]
		buffer.WriteByte(ch)
		if (i+1)%76 == 0 {
			buffer.WriteByte('\n')
		}
	}
	buffer.WriteByte('\n')
	return buffer.Bytes()
}

//go:embed icon.json
var icon_json string

func IconDetect(Url string) (string, string, error) {
	req, _ := http.NewRequest(http.MethodGet, Url, nil)
	SetHeaders(req)
	resp, err := Client.Do(req)

	if err != nil {
		logRequestError(Url, err)
		return "", "", err
	}
	defer resp.Body.Close()
	bodyBytes, _ := io.ReadAll(resp.Body)
	ico := Mmh3Hash32(StandBase64(bodyBytes))

	hash := md5.Sum(bodyBytes)
	hunterIco := hex.EncodeToString(hash[:])

	return ico, hunterIco, nil
}

// IconDetectAuto accepts either a direct favicon URL or a page URL and extracts the rel=icon href.
func IconDetectAuto(target string) (string, string, error) {
	u, err := url.Parse(target)
	if err != nil {
		return "", "", err
	}
	if strings.HasSuffix(strings.ToLower(u.Path), ".ico") {
		return IconDetect(target)
	}
	// 直接复用页面解析逻辑（单次请求页面获取 rel=icon）
	iconURL, err := FindFaviconURL(target)
	if err != nil {
		return "", "", err
	}
	return IconDetect(iconURL)
}

func FindFaviconURL(urlStr string) (string, error) {
	// 解析基准URL
	baseURL, err := url.Parse(urlStr)
	if err != nil {
		return "", err
	}

	// 获取HTML内容
	req, _ := http.NewRequest(http.MethodGet, urlStr, nil)
	SetHeaders(req)
	resp, err := Client.Do(req)
	if err != nil {
		logRequestError(urlStr, err)
		return "", err
	}
	defer resp.Body.Close()
	// 从响应中创建goquery文档
	doc, err := goquery.NewDocumentFromReader(resp.Body)
	if err != nil {
		return "", err
	}
	// 创建正则表达式，模糊匹配rel属性值
	r := regexp.MustCompile(`icon`)

	// 查找匹配的<link>标签
	var faviconURL string
	doc.Find("link").Each(func(i int, s *goquery.Selection) {
		// 检查rel属性值是否匹配正则表达式
		rel, exists := s.Attr("rel")
		if exists && r.MatchString(rel) {
			// 提取href属性值
			href, exists := s.Attr("href")
			if exists {
				// 检查是否是绝对路径
				if strings.HasPrefix(href, "//") {
					faviconURL = baseURL.Scheme + ":" + href
				} else if isAbsoluteURL(href) {
					faviconURL = href
				} else {
					// 解析相对路径并构建完整URL
					faviconURL = baseURL.ResolveReference(&url.URL{Path: href}).String()
				}
			}
		}
	})

	if faviconURL == "" {
		return "", errors.New("favicon url not found, might used javascript, please find it manually and use `godscan icon -u` to calculate it")
	}

	return faviconURL, nil
}

// FindFaviconURLFromHTML tries to extract favicon from provided HTML; does not issue new GET.
func FindFaviconURLFromHTML(rootURL string, htmlBody string) (string, error) {
	baseURL, err := url.Parse(rootURL)
	if err != nil {
		return "", err
	}
	doc, err := goquery.NewDocumentFromReader(strings.NewReader(htmlBody))
	if err != nil {
		return "", err
	}
	r := regexp.MustCompile(`icon`)
	var faviconURL string
	doc.Find("link").Each(func(i int, s *goquery.Selection) {
		rel, exists := s.Attr("rel")
		if exists && r.MatchString(rel) {
			href, exists := s.Attr("href")
			if exists {
				if strings.HasPrefix(href, "//") {
					faviconURL = baseURL.Scheme + ":" + href
				} else if isAbsoluteURL(href) {
					faviconURL = href
				} else {
					faviconURL = baseURL.ResolveReference(&url.URL{Path: href}).String()
				}
			}
		}
	})
	if faviconURL == "" {
		return "", errors.New("favicon url not found in html")
	}
	return faviconURL, nil
}

// 检查URL是否是绝对路径
func isAbsoluteURL(urlStr string) bool {
	u, err := url.Parse(urlStr)
	if err != nil {
		return false
	}
	return u.IsAbs()
}

func ApiDeDuplicate(RootPath string, directory string) {
	rawApi := FileReadLine(directory + "api_raw.txt")
	fullPaths := []string{}
	if len(rawApi) > 0 {
		FileWrite(directory+"api_unique.txt", "%s", strings.Join(rawApi, "\n"))
		for _, raw := range rawApi {
			fullPaths = append(fullPaths, RootPath+raw)
		}
		FileWrite(directory+"api_unique_full.txt", "==== Try: dirbrute --url-file %s\n", directory+"api_unique_full.txt")
		FileWrite(directory+"api_unique_full.txt", "%s\n", strings.Join(fullPaths, "\n"))
	}
}

func PrintFinger(Url string, Depth int) {
	summary := FingerSummary(Url, Depth, nil)
	if summary.URL == "" {
		return
	}
	table := prettytable.NewWriter()
	table.SetOutputMirror(os.Stdout)
	table.AppendHeader(prettytable.Row(StringListToInterfaceList(common.TableHeader)))
	table.SetStyle(prettytable.StyleRounded)
	table.SetColumnConfigs([]prettytable.ColumnConfig{
		{Number: 1, WidthMax: 42},
		{Number: 2, WidthMax: 32},
		{Number: 3, WidthMax: 36},
		{Number: 4, WidthMax: 24},
		{Number: 5, WidthMax: 8, Transformer: StatusColorTransformer},
		{Number: 8, WidthMax: 28},
	})
	row := []string{summary.URL, summary.Title, summary.Finger, summary.ContentType, strconv.Itoa(summary.Status), "", strconv.Itoa(summary.Length), summary.Keyword, summary.SimHash}
	AddDataToTable(table, row)
	if table.Length() >= 1 {
		table.Render()
	}
}

// FingerSummary scans a target and returns a compact per-host summary while still writing detailed CSV files.
func FingerSummary(Url string, Depth int, db *sql.DB) SpiderSummary {
	out := SpiderSummary{URL: Url, Status: -1}
	Host, err := url.Parse(Url)
	if err != nil {
		Error("%s", err)
		out.Err = err
		return out
	}
	RootPath := Host.Scheme + "://" + Host.Hostname()
	if Host.Port() != "" {
		RootPath = RootPath + ":" + Host.Port()
	}

	// 首页
	FirstUrl := RootPath + Host.Path
	fres := FingerScan(FirstUrl, http.MethodGet, true)
	out.Status = fres.Status
	out.Finger = fres.Finger
	out.Title = fres.Title
	out.ContentType = fres.ContentType
	out.Length = len(fres.Body)
	_ = SavePageSnapshot(db, PageSnapshot{
		RootURL:     RootPath,
		URL:         FirstUrl,
		Status:      fres.Status,
		ContentType: fres.ContentType,
		Headers:     fres.HeadersJSON,
		Body:        string(fres.Body),
		Length:      len(fres.Body),
	})

	if fres.Status != -1 {
		result := CheckFinger(fres.Finger, fres.Title, Url, fres.ContentType, "", fres.Body, fres.Status)
		if len(result) > 0 {
			WriteToCsv("finger.csv", result)
			out.Finger = result[2]
			out.ContentType = result[3]
			out.Status, _ = strconv.Atoi(result[4])
			out.Length, _ = strconv.Atoi(result[6])
			out.Keyword = result[7]
			out.SimHash = result[8]
		}
	}

	// 当首个 GET 无有效指纹时，再补充 POST/404 探针以提高识别率
	if out.Finger == "" || out.Finger == common.NoFinger {
		SecondUrl := RootPath + "/xxxxxx"
		fres2 := FingerScan(SecondUrl, http.MethodPost, true)
		if fres2.Status != -1 {
			result := CheckFinger(fres2.Finger, fres2.Title, Url, fres2.ContentType, "", fres2.Body, fres2.Status)
			if len(result) > 0 && result[2] != common.NoFinger {
				WriteToCsv("finger.csv", result)
				out.Finger = result[2]
				out.Title = fres2.Title
				out.ContentType = fres2.ContentType
				out.Status, _ = strconv.Atoi(result[4])
				out.Length, _ = strconv.Atoi(result[6])
				out.Keyword = result[7]
				out.SimHash = result[8]
			}
		}
	}

	// 仅用已获取的首页 HTML 提取 favicon，找不到则放弃，避免额外请求
	if iconURL, iconErr := FindFaviconURLFromHTML(RootPath, string(fres.Body)); iconErr == nil {
		if fofaHash, hunterHash, errHash := IconDetect(iconURL); errHash == nil {
			out.IconHash = fmt.Sprintf("fofa: icon_hash=\"%s\"\nhunter: web.icon=\"%s\"", fofaHash, hunterHash)
			var icon_hash_map map[string]interface{}
			json.Unmarshal([]byte(icon_json), &icon_hash_map)
			if tmp := icon_hash_map[fofaHash]; tmp != nil {
				Debug("icon_url=\"%s\" icon_finger=\"%s\"", iconURL, tmp)
			}
		} else {
			Debug("%s", errHash)
		}
	}
	// 爬虫递归爬
	myMap := mapset.NewSet()
	sourceMapSeen := mapset.NewSet()
	host, err := url.Parse(RootPath)
	if err != nil {
		Error("%s", err)
		out.Err = err
		return out
	}
	directory := fmt.Sprintf("%s/%s/spider/", time.Now().Format("2006-01-02"), host.Hostname()+"_"+host.Port())
	apiCounter := 0
	err = Spider(RootPath, Url, Depth, directory, myMap, sourceMapSeen, &apiCounter, db, &prefetchedPage{
		body:        string(fres.Body),
		contentType: fres.ContentType,
	})
	if err != nil {
		Error("%s", err)
		out.Err = err
		return out
	}
	out.SaveDir = directory
	out.ApiCount = apiCounter
	ApiDeDuplicate(RootPath, directory)

	var myList []string
	for item := range myMap.Iter() {
		myList = append(myList, item.(string))
	}
	out.UrlCount = len(myList)
	cdnHosts := collectCDNHosts(myList)
	out.CDNCount = len(cdnHosts)
	out.CDNHosts = strings.Join(cdnHosts, ",")
	if len(cdnHosts) > 0 {
		FileWrite(directory+"cdn_hosts.txt", "%s\n", strings.Join(cdnHosts, "\n"))
		SaveCDNHosts(db, RootPath, cdnHosts)
	}
	FileWrite(directory+"spider.log", "%s\n", strings.Join(myList, "\n"))
	return out
}

func collectCDNHosts(urls []string) []string {
	candidates := []string{
		"aliyuncs.com", "alicdn.com", "qiniucdn.com", "qiniu.com", "myqcloud.com", "tencentcs.com", "ksyuncs.com", "bcebos.com", "cloudfront.net", "bitly.com",
	}
	seen := make(map[string]struct{})
	for _, raw := range urls {
		u, err := url.Parse(raw)
		if err != nil {
			continue
		}
		host := strings.ToLower(u.Hostname())
		for _, c := range candidates {
			if strings.HasSuffix(host, c) {
				seen[host] = struct{}{}
				break
			}
		}
	}
	out := make([]string, 0, len(seen))
	for h := range seen {
		out = append(out, h)
	}
	sort.Strings(out)
	return out
}
