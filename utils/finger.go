package utils

import (
	"bytes"
	_ "embed"
	"errors"
	"fmt"
	"hash"
	"io"
	"os"
	"path"
	"sort"
	"sync"

	"github.com/spf13/viper"

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
		return "", "", "", err
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

func parseHost(input string) string {
	parsedURL, err := url.Parse(input)
	if err != nil {
		return ""
	}
	return parsedURL.Hostname()
}

func uselessUrl(url string) bool {
	ignore := []string{".min.js", ".png", ".jpeg", ".jpg", ".gif", ".bmp", "chunk-vendors", ".vue"}
	for _, ign := range ignore {
		if strings.Contains(url, ign) {
			return true
		}
	}
	return false
}

func parseDir(fullPath string) []string {
	// 去除末尾的斜杠
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
		dirs = append(dirs, dir)
	}
	return dirs
}

func parseVueUrl(Url string, RootPath string, doc string) {
	color.HiYellow("->[*] [%s] Api Path", Url)
	ApiReg := regexp.MustCompile(`"(?P<path>/.*?)"`)
	ApiResultTuple := ApiReg.FindAllStringSubmatch(strings.ReplaceAll(doc, "\t", ""), -1)
	ApiResult := []string{}

	for _, tmp := range ApiResultTuple {
		if uselessUrl(tmp[1]) {
			continue
		}
		ApiResult = append(ApiResult, viper.GetString("ApiPrefix")+tmp[1])
	}
	ApiResult = removeDuplicatesString(ApiResult)
	fmt.Println(strings.Join(ApiResult, "\n"))

	subdir := []string{}
	matches := []string{}

	for _, apiPath := range ApiResult {
		matches = append(matches, parseDir(apiPath)...)
	}

	matches = removeDuplicatesString(matches)
	for _, match := range matches {
		normalizeUrl := Normalize(match, RootPath)
		subdir = append(subdir, normalizeUrl)
	}
	subdir = removeDuplicatesString(subdir)
	fmt.Println(color.YellowString("->[*] sub-directory"))
	fmt.Println(color.BlueString(strings.Join(subdir, "\n")))

	var wg sync.WaitGroup
	result := []string{}
	for _, line := range subdir {
		wg.Add(1)
		go func(url string) {
			defer wg.Done()
			result = append(result, DirBrute(url)...)
		}(line)
	}
	wg.Wait()
	if len(result) > 0 {
		filename := fmt.Sprintf("brute_%s.log", parseHost(RootPath))
		Success("More at ./%s", filename)
		os.WriteFile(filename, []byte(strings.Join(result, "\n")), 0644)
	}
}

func Spider(RootPath string, Url string, depth int, myMap map[int][]string) error {
	if depth == 0 || uselessUrl(Url) {
		return nil
	}
	myMap[depth] = append(myMap[depth], Url)

	req, err := http.NewRequest(http.MethodGet, Url, nil)
	if err != nil {
		return err
	}
	req.Header.Set("User-Agent", viper.GetString("DefaultUA"))
	resp, err := Client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	doc, err := goquery.NewDocumentFromReader(resp.Body)
	if err != nil {
		Error("%s", err)
		return err
	}

	// 如果是vue.js app.xxxxxxxx.js 识别其中的api接口
	if strings.HasSuffix(Url, ".js") && IsVuePath(Url) {
		parseVueUrl(Url, RootPath, doc.Text())
	} else {
		// 敏感信息搜集
		html, err := doc.Html()
		if err != nil {
			return err
		}
		SensitiveInfoCollect(Url, html)

		// a标签
		doc.Find("a").Each(func(i int, selector *goquery.Selection) {
			href, _ := selector.Attr("href")
			normalizeUrl := Normalize(href, RootPath)
			if normalizeUrl != "" && !in(myMap[depth-1], normalizeUrl) {
				Spider(RootPath, normalizeUrl, depth-1, myMap)
			}
		})
		// iframe, script 标签
		doc.Find("script, iframe").Each(func(i int, selector *goquery.Selection) {
			src, _ := selector.Attr("src")
			normalizeUrl := Normalize(src, RootPath)
			if normalizeUrl != "" && !in(myMap[depth-1], normalizeUrl) {
				Spider(RootPath, normalizeUrl, depth-1, myMap)
			}
		})
	}
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
	req, _ := http.NewRequest(http.MethodGet, Url, nil)
	req.Header.Set("User-Agent", viper.GetString("DefaultUA"))
	resp, err := Client.Do(req)

	if err != nil {
		Error("%s", err)
		return "", err
	}
	defer resp.Body.Close()
	bodyBytes, _ := io.ReadAll(resp.Body)
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
	res, _, _, _ := FingerScan(FirstUrl)
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
		if len(url) > 1 {
			filename := fmt.Sprintf("%s_%d_%d.log", Host.Hostname(), depth, len(url))
			Success("Depth: %d total=%d, More at ./%s", depth, len(url), filename)
			os.WriteFile(filename, []byte(strings.Join(url, "\n")), 0644)
		}
	}
}
