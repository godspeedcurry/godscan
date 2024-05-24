package utils

import (
	"bytes"
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
	"github.com/olekukonko/tablewriter"
	"github.com/spf13/viper"

	"net/http"
	"net/url"
	"regexp"
	"strings"

	b64 "encoding/base64"
	"encoding/json"

	"github.com/PuerkitoBio/goquery"

	mapset "github.com/deckarep/golang-set"
	"github.com/fatih/color"
	"github.com/twmb/murmur3"
)

var sensitiveUrl sync.Map

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
	if err != nil {
		Error("%s", err)
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
		fmt.Println(Url + "\n" + data)
	}
}

func uselessUrl(Url string, Depth int) bool {
	if Depth == 0 || Url == "" || !isValidUrl(Url) {
		return true
	}
	ignore := []string{".min.js", ".png", ".jpeg", ".jpg", ".gif", ".bmp", ".vue", ".css", ".ico", ".svg"}
	for _, ign := range ignore {
		if strings.Contains(Url, ign) {
			return true
		}
	}
	return false
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
	arr := []string{"alicdn.com", "163.com", "nginx.com", "qq.com", "amap.com", "cnzz.com", "github.com"}
	for _, key := range arr {
		if strings.Contains(Url, key) {
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
	ip := net.ParseIP(parsedURL.Hostname())
	if ip != nil {
		return !ip.IsPrivate()
	}
	return true
}

func ImportantApiJudge(ApiResult string, Url string) {
	for _, key := range common.ImportantApi {
		if strings.Contains(ApiResult, key) {
			Fatal("Import Api found " + key)
			if key == "/api/blade-user" {
				Fatal("Might related to SpringBlade CVE-2021-44910")
				FileWrite("cve.log", "[%s] Might related to SpringBlade CVE-2021-44910", Url)
			}
		}
	}
}

func parseVueUrl(Url string, RootPath string, doc string, filename string) {
	file, err := os.OpenFile(filename+".sensi", os.O_APPEND|os.O_WRONLY|os.O_CREATE, 0644)
	if err != nil {
		Error("%s", err)
		return
	}

	ApiReg := regexp.MustCompile(`"(?P<path>/[\w/\-_=@\?\:]+?)"`)

	ApiResultTuple := ApiReg.FindAllStringSubmatch(strings.ReplaceAll(doc, "\\", ""), -1)
	ApiResult := []string{}

	for _, tmp := range ApiResultTuple {
		ApiResult = append(ApiResult, viper.GetString("ApiPrefix")+tmp[1])
	}
	ApiResult = RemoveDuplicatesString(ApiResult)
	ApiResultLen := len(ApiResult)
	if ApiResultLen > 0 {
		file.WriteString("->[*] [" + Url + "] Api Path\n")
		Success("->[*] [" + Url + "] Api Path")
		if ApiResultLen > 50 {
			var tmpResult1 = strings.Join(ApiResult[:50], "\n")
			Success(tmpResult1)
			file.WriteString(tmpResult1 + "\n")
		} else {
			var tmpResult2 = strings.Join(ApiResult, "\n")
			ImportantApiJudge(tmpResult2, Url)
			Success(tmpResult2)
			file.WriteString(tmpResult2 + "\n")
		}
	}

	subdir := []string{}
	matches := []string{}

	for _, apiPath := range ApiResult {
		matches = append(matches, parseDir(apiPath, 2)...)
	}

	matches = RemoveDuplicatesString(matches)
	for _, match := range matches {
		normalizeUrl := Normalize(match, RootPath)
		if !isValidUrl(normalizeUrl) {
			continue
		}
		subdir = append(subdir, normalizeUrl)
	}
	subdir = RemoveDuplicatesString(subdir)
	if len(subdir) > 0 {
		file.WriteString("->[*] sub-directory\n")
		file.WriteString(strings.Join(subdir, "\n") + "\n")

	}

	var wg sync.WaitGroup
	multiWriter := io.MultiWriter(os.Stdout, file)
	table := tablewriter.NewWriter(multiWriter)

	// 创建表格
	table.SetHeader([]string{"Url", "Title", "Finger", "Content-Type", "StatusCode", "Length"})

	cnt := 0

	for _, line := range subdir {
		for _, dir := range []string{".git/config", "swagger-resources", "v2/api-docs", ""} {
			cnt += 1
			if cnt >= 50 {
				continue
			}
			wg.Add(1)
			go func(url string, dir string) {
				defer wg.Done()
				AddDataToTable(table, DirBrute(url, dir))
			}(line, dir)
		}
	}
	wg.Wait()
	if table.NumLines() >= 1 {
		table.Render()
	}
	if _, ok := sensitiveUrl.Load(Url); !ok {
		sensitiveUrl.Store(Url, true)
		SensitiveInfoCollect(Url, doc)
	}

}

func Spider(RootPath string, Url string, depth int, myMap mapset.Set) error {
	if uselessUrl(Url, depth) {
		return nil
	}
	myMap.Add(Url)
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
	host, err := url.Parse(RootPath)
	if err != nil {
		Error("%s", err)
		return err
	}
	filename := fmt.Sprintf("%s/%s.log", time.Now().Format("2006-01-02"), host.Hostname())
	// 如果是vue.js app.xxxxxxxx.js 识别其中的api接口
	if IsVuePath(Url) {
		fmt.Println(Url)
		bufStr := ""
		buf := make([]byte, 4096)
		for {
			n, err := resp.Body.Read(buf)
			if err == io.EOF {
				bufStr += string(buf[:n])
				break
			}
			bufStr += string(buf[:n])
		}
		parseVueUrl(Url, RootPath, bufStr, filename)
	} else {
		doc, err := goquery.NewDocumentFromReader(resp.Body)
		if err != nil {
			Error("%s", err)
			return err
		}
		// 敏感信息搜集
		html, err := doc.Html()
		if err != nil {
			Error("%s", err)
			return err
		}
		if _, ok := sensitiveUrl.Load(Url); !ok {
			sensitiveUrl.Store(Url, true)
			SensitiveInfoCollect(Url, html)
		}

		// a, link 标签
		doc.Find("a, link").Each(func(i int, selector *goquery.Selection) {
			href, _ := selector.Attr("href")
			normalizeUrl := Normalize(href, RootPath)
			if normalizeUrl != "" && !myMap.Contains(normalizeUrl) {
				Spider(RootPath, normalizeUrl, depth-1, myMap)
			}
		})
		// iframe, script 标签
		doc.Find("script, iframe").Each(func(i int, selector *goquery.Selection) {
			src, _ := selector.Attr("src")
			normalizeUrl := Normalize(src, RootPath)

			if normalizeUrl != "" && !myMap.Contains(normalizeUrl) {
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
		return "", errors.New("favicon url not found, might used javascript, please find it manually and use `godscan icon -u` to calculate it")
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
	Host, err := url.Parse(Url)
	if err != nil {
		Error("%s", err)
		return
	}
	RootPath := Host.Scheme + "://" + Host.Hostname()
	if Host.Port() != "" {
		RootPath = RootPath + ":" + Host.Port()
	}

	// 首页
	FirstUrl := RootPath + Host.Path

	finger, server, title, contentType, respBody, statusCode := FingerScan(FirstUrl, http.MethodGet)

	if statusCode != -1 {
		result := CheckFinger(finger, title, Url, contentType, respBody, statusCode)
		if len(result) > 0 {
			WriteToCsv("finger.csv", result)
		}
		Info("%s [%s] [%s] [%s] [%d]", Url, finger, server, title, statusCode)
	}

	// 构造404 + POST
	SecondUrl := RootPath + "/xxxxxx"
	finger, server, title, contentType, respBody, statusCode = FingerScan(SecondUrl, http.MethodPost)
	if statusCode != -1 {
		result := CheckFinger(finger, title, Url, contentType, respBody, statusCode)
		if len(result) > 0 {
			WriteToCsv("finger.csv", result)
		}
		Info("%s [%s] [%s] [%s] [%d]", SecondUrl, finger, server, title, statusCode)
	}

	IconUrl, err := FindFaviconURL(RootPath)
	if err == nil {
		IconDetect(IconUrl)
	} else {
		Error("%s", err)
	}
	// 爬虫递归爬
	myMap := mapset.NewSet()
	err = Spider(RootPath, Url, Depth, myMap)
	if err != nil {
		Error("%s", err)
		return
	}

	filename := fmt.Sprintf("%s/%s.log", time.Now().Format("2006-01-02"), Host.Hostname())
	Success("🌲🌲🌲 More info at ./%s", filename)
	var myList []string
	for item := range myMap.Iter() {
		myList = append(myList, item.(string))
	}
	FileWrite(filename, strings.Join(myList, "\n")+"\n")
}
