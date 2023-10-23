package utils

import (
	"bytes"
	_ "embed"
	"errors"
	"fmt"
	"hash"
	"io/ioutil"
	"sort"

	"github.com/spf13/viper"

	"net"
	"net/http"
	"net/url"
	"regexp"
	"strings"

	b64 "encoding/base64"
	"encoding/json"

	"github.com/PuerkitoBio/goquery"

	"github.com/fatih/color"
	"github.com/twmb/murmur3"
)

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
	req.Header.Set("User-Agent", viper.GetString("DefaultUA"))
	resp, err := Client.Do(req)
	if err != nil {
		return "", "", "", err
	}
	defer resp.Body.Close()
	doc, err := goquery.NewDocumentFromReader(resp.Body)
	title := doc.Find("title").Text()
	if err != nil {
		Fatal("%s", err)
	}
	ServerValue := resp.Header["Server"]
	Status := resp.Status
	retServerValue := ""
	if len(ServerValue) != 0 {
		retServerValue = ServerValue[0]
	}
	return retServerValue, Status, title, nil
}

//go:embed finger.txt
var fingers string

func GetFingerList() []string {
	return strings.Split(fingers, "\r\n")
}

func FindKeyWord(data string) []string {
	keywords := []string{}
	m := make(map[string]int)
	fingerList := GetFingerList()
	for _, finger := range fingerList {
		if strings.Contains(data, finger) {
			cnt := strings.Count(data, finger)
			keywords = append(keywords, finger)
			m[finger] = cnt
		}
	}
	return keywords
}

func IsVuePath(Path string) bool {
	reg := regexp.MustCompile(`(app|index|config|main|chunk)`)
	res := reg.FindAllString(Path, -1)
	return len(res) > 0
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
		fmt.Println(Url + "\n" + data + "\n")
	}
}

func parseHost(RootPath string) (string, error) {
	urlStruct, err := url.Parse(RootPath)
	if err != nil {
		return "", err
	}
	host, _, _ := net.SplitHostPort(urlStruct.Host)
	return host, nil
}

func uselessUrl(url string) bool {
	return strings.Contains(url, ".min.js") || strings.Contains(url, ".ico") || strings.Contains(url, "chunk-vendors")
}

func Spider(RootPath string, Url string, depth int, myMap map[int][]string) error {
	myMap[depth] = append(myMap[depth], Url)
	host, _ := parseHost(RootPath)
	if !strings.Contains(Url, host) || depth == 0 || uselessUrl(Url) {
		return nil
	}

	req, _ := http.NewRequest(http.MethodGet, Url, nil)
	req.Header.Set("User-Agent", viper.GetString("DefaultUA"))
	resp, err := Client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	doc, err := goquery.NewDocumentFromReader(resp.Body)
	if err != nil {
		Error("%s", err)
		return nil
	}

	keywords := FindKeyWord(doc.Text())

	//正则提取版本
	VersionReg := regexp.MustCompile(`(?i)(version|ver|v|版本)[ =:]{0,2}(\d+)(\.[0-9a-z]+)*`)

	VersionResult := VersionReg.FindAllString(strings.ReplaceAll(doc.Text(), "\t", ""), -1)

	VersionResultNotDupplicated := removeDuplicatesString(VersionResult)

	//正则提取注释
	AnnotationReg := regexp.MustCompile("/\\*[\u0000-\uffff]{1,300}?\\*/")
	AnnotationResult := AnnotationReg.FindAllString(strings.ReplaceAll(doc.Text(), "\t", ""), -1)
	if len(AnnotationResult) > 0 {
		for _, Annotation := range AnnotationResult {
			HighLight(Annotation, VersionResultNotDupplicated, keywords, Url)
		}
	}

	// 如果是vue.js app.xxxxxxxx.js 识别其中的api接口
	if IsVuePath(Url) {
		color.HiYellow("->[*] Api Path")
		ApiReg := regexp.MustCompile(`"(?P<path>/.*?)"`)
		ApiResultTuple := ApiReg.FindAllStringSubmatch(strings.ReplaceAll(doc.Text(), "\t", ""), -1)
		ApiResult := []string{}

		for _, tmp := range ApiResultTuple {
			ApiResult = append(ApiResult, viper.GetString("ApiPrefix")+tmp[1])
		}
		ApiResult = removeDuplicatesString(ApiResult)
		fmt.Println(strings.Join(ApiResult, "\n"))

	}

	// 敏感信息搜集
	html, _ := doc.Html()
	SensitiveInfoCollect(html)

	// a标签
	doc.Find("a").Each(func(i int, selector *goquery.Selection) {
		href, _ := selector.Attr("href")
		normalizeUrl := Normalize(href, RootPath)
		if normalizeUrl != "" && !in(myMap[depth-1], (normalizeUrl)) {
			Spider(RootPath, normalizeUrl, depth-1, myMap)
		}
	})
	// iframe, script 标签
	doc.Find("script, iframe").Each(func(i int, selector *goquery.Selection) {
		src, _ := selector.Attr("src")
		normalizeUrl := Normalize(src, RootPath)
		if normalizeUrl != "" && !in(myMap[depth-1], (normalizeUrl)) {
			Spider(RootPath, normalizeUrl, depth-1, myMap)
		}
	})
	return nil
}

func DisplayHeader(Url string, Method string) {
	ServerHeader, Status, Title, err := HttpGetServerHeader(Url, true, Method)
	if err != nil {
		Error("Error: %s", err)
	} else {
		Info("[%s] [%s] [%s] [%s] [%s]", Url, Method, ServerHeader, Status, Title)
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

func IconDetect(Url string) (string, error) {
	InitHttp()
	req, _ := http.NewRequest(http.MethodGet, Url, nil)
	req.Header.Set("User-Agent", viper.GetString("DefaultUA"))
	resp, err := Client.Do(req)

	if err != nil {
		Error("%s", err)
		return "", err
	}
	defer resp.Body.Close()
	bodyBytes, _ := ioutil.ReadAll(resp.Body)
	ico := Mmh3Hash32(StandBase64(bodyBytes))
	Info("icon_url=\"%s\" icon_hash=\"%s\" %d", Url, ico, resp.StatusCode)
	var icon_hash_map map[string]interface{}
	json.Unmarshal([]byte(icon_json), &icon_hash_map)
	tmp := icon_hash_map[ico]
	if tmp != nil {
		Success("icon_url=\"%s\" icon_finger=\"%s\"", Url, tmp)
	}
	return "", nil
}

func FindFaviconURL(urlStr string) (string, error) {
	// 解析基准URL
	InitHttp()
	baseURL, err := url.Parse(urlStr)
	if err != nil {
		return "", err
	}

	// 获取HTML内容
	req, _ := http.NewRequest(http.MethodGet, urlStr, nil)
	req.Header.Set("User-Agent", viper.GetString("DefaultUA"))
	resp, err := Client.Do(req)
	if err != nil {
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
				if isAbsoluteURL(href) {
					faviconURL = href
				} else {
					// 解析相对路径并构建完整URL
					faviconURL = baseURL.ResolveReference(&url.URL{Path: href}).String()
				}
			}
		}
	})

	if faviconURL == "" {
		return "", errors.New("Favicon URL not found, might used javascript, please find it manually and use `-ico url` to calculate it")
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

func PrintFinger(Url string, Depth int) {
	InitHttp()
	if !strings.HasPrefix(Url, "http") {
		Url = "http://" + Url
	}
	Host, _ := url.Parse(Url)
	RootPath := Host.Scheme + "://" + Host.Hostname()
	if Host.Port() != "" {
		RootPath = RootPath + ":" + Host.Port()
	}

	// 首页
	FirstUrl := RootPath + Host.Path
	res, _, _ := FingerScan(FirstUrl)
	if res != "" {
		Info(Url + " " + res)
	}

	DisplayHeader(FirstUrl, http.MethodGet)

	// 构造404
	SecondUrl := RootPath + "/xxxxxx"
	DisplayHeader(SecondUrl, http.MethodGet)

	// 构造POST
	ThirdUrl := RootPath
	DisplayHeader(ThirdUrl, http.MethodPost)

	IconUrl, err := FindFaviconURL(RootPath)
	if err == nil {
		IconDetect(IconUrl)
	} else {
		Error("%s", err)
		return
	}
	// 爬虫递归爬
	myMap := make(map[int][]string)

	err = Spider(RootPath, Url, Depth, myMap)
	if err != nil {
		Error("%s", err)
		return
	}
	for depth, url := range myMap {
		url := removeDuplicatesString(url)
		sort.Strings(url)
		if len(url) > 0 {
			filename := fmt.Sprintf("%s_%d_%s.log", Host.Hostname(), depth, RandomString(4))
			Success("Depth: %d total=%d, More at ./%s", depth, len(url), filename)
			ioutil.WriteFile(filename, []byte(strings.Join(url, "\n")), 0644)
		}
	}
}
