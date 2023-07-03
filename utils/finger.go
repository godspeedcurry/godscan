package utils

import (
	"bytes"
	_ "embed"
	"fmt"
	"hash"
	"io/ioutil"
	"log"
	"main/common"
	"net"
	"net/http"
	"net/url"
	"regexp"
	"strconv"
	"strings"

	b64 "encoding/base64"
	"encoding/json"

	"github.com/PuerkitoBio/goquery"
	mapset "github.com/deckarep/golang-set"

	"github.com/fatih/color"
	"github.com/scylladb/termtables"
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
	req, _ := http.NewRequest(Method, Url, nil)

	if Method == http.MethodPost {
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	}
	req.Header.Set("User-Agent", "Mozilla/5.0 (Macintosh; Intel Mac OS X 100) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/1.0.5005.61 Safari/537.36")
	resp, err := Client.Do(req)
	if err != nil {
		return "", "", "", err
	}
	defer resp.Body.Close()
	doc, err := goquery.NewDocumentFromReader(resp.Body)
	title := doc.Find("title").Text()
	if err != nil {
		log.Fatal(err)
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

func GenColumn(column int) []interface{} {
	var headers []interface{} = make([]interface{}, column)
	for i := 0; i < (column >> 1); i++ {
		headers[i<<1] = "Key" + strconv.Itoa(i+1)
		headers[(i<<1)+1] = "Value" + strconv.Itoa(i+1)
	}
	return headers
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
	mSorted := mysort(m)
	table := termtables.CreateTable()
	maxColumn := Min(10, len(mSorted)<<1)
	if maxColumn == 0 {
		return keywords
	}
	table.AddHeaders(GenColumn(maxColumn)...)
	tmpList := []string{}
	cnt := 0

	for _, tmp := range mSorted {
		tmpList = append(tmpList, tmp.Key)
		tmpList = append(tmpList, strconv.Itoa(tmp.Value))
		cnt++
		if cnt%(maxColumn>>1) == 0 {
			table.AddRow(StringListToInterfaceList(tmpList[:maxColumn])...)
			tmpList = []string{}
		}
	}
	if cnt%(maxColumn>>1) != 0 {
		tmpList = append(tmpList, make([]string, maxColumn)...)
		table.AddRow(StringListToInterfaceList(tmpList[:maxColumn])...)
	}
	color.Cyan("%s\n", table.Render())
	return keywords
}

func IsVuePath(Path string) bool {
	reg := regexp.MustCompile(`app\.[0-9a-z]+\.js`)
	res := reg.FindAllString(Path, -1)
	return len(res) > 0
}

func HighLight(data string, keywords []string, fingers []string) {
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
		fmt.Println(data)
	}
}

func Spider(RootPath string, Url string, depth int, s1 mapset.Set) (string, error) {
	url_struct, _ := url.Parse(RootPath)
	host, _, _ := net.SplitHostPort(url_struct.Host)
	if !strings.Contains(Url, host) {
		fmt.Printf("[Depth %d] %s\n", depth, Url)
		s1.Add(Url)
		return "", nil
	} else if depth == 0 || strings.Contains(Url, ".min.js") || strings.Contains(Url, ".ico") || strings.Contains(Url, "chunk-vendors") {
		return "", nil
	}
	fmt.Printf("[Depth %d] %s\n", depth, Url)
	s1.Add(Url)
	req, _ := http.NewRequest(http.MethodGet, Url, nil)
	req.Header.Set("User-Agent", "Mozilla/5.0 (Macintosh; Intel Mac OS X 100) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/1.0.5005.61 Safari/537.36")
	resp, err := Client.Do(req)
	if err != nil {
		fmt.Println(err)
		return "", err
	}
	defer resp.Body.Close()
	doc, _ := goquery.NewDocumentFromReader(resp.Body)
	keywords := FindKeyWord(doc.Text())

	//正则提取版本
	VersionReg := regexp.MustCompile(`(?i)(version|ver|v|版本)[ =:]{0,2}(\d+)(\.[0-9a-z]+)*`)

	VersionResult := VersionReg.FindAllString(strings.ReplaceAll(doc.Text(), "\t", ""), -1)
	var VersionResultNotDupplicated interface{}
	if len(VersionResult) > 0 {
		VersionResultNotDupplicated, _ = RemoveDuplicateElement(VersionResult)
	}

	//正则提取注释
	AnnotationReg := regexp.MustCompile("/\\*[\u0000-\uffff]{1,300}?\\*/")
	AnnotationResult := AnnotationReg.FindAllString(strings.ReplaceAll(doc.Text(), "\t", ""), -1)
	if len(AnnotationResult) > 0 {
		fmt.Println("[*] 注释部分 && 版本识别")
		for _, Annotation := range AnnotationResult {
			if VersionResultNotDupplicated == nil {
				HighLight(Annotation, []string{}, keywords)
			} else {
				HighLight(Annotation, VersionResultNotDupplicated.([]string), keywords)
			}
		}
	}

	// 如果是vue.js app.xxxxxxxx.js 识别其中的api接口
	if IsVuePath(Url) {
		fmt.Println("[*] Api Path")
		ApiReg := regexp.MustCompile(`path:"(?P<path>.*?)"`)
		ApiResult := ApiReg.FindAllStringSubmatch(strings.ReplaceAll(doc.Text(), "\t", ""), -1)
		if len(ApiResult) > 0 {
			for _, tmp := range ApiResult {
				fmt.Println(RootPath + "/" + tmp[1])
			}
		}
	}

	// 敏感信息搜集
	html, _ := doc.Html()
	SensitiveInfoCollect(html)

	// a标签
	doc.Find("a").Each(func(i int, a *goquery.Selection) {
		href, _ := a.Attr("href")
		normalizeUrl := Normalize(href, RootPath)
		if normalizeUrl != "" && !s1.Contains(normalizeUrl) {
			Spider(RootPath, normalizeUrl, depth-1, s1)
		}
	})
	// script 标签
	doc.Find("script").Each(func(i int, script *goquery.Selection) {
		src, _ := script.Attr("src")
		normalizeUrl := Normalize(src, RootPath)
		if normalizeUrl != "" && !s1.Contains(normalizeUrl) {
			Spider(RootPath, normalizeUrl, depth-1, s1)
		}
	})

	//iframe 标签
	doc.Find("iframe").Each(func(i int, iframe *goquery.Selection) {
		src, _ := iframe.Attr("src")
		normalizeUrl := Normalize(src, RootPath)
		if normalizeUrl != "" && !s1.Contains(normalizeUrl) {
			Spider(RootPath, normalizeUrl, depth-1, s1)
		}
	})
	return "", nil
}

func DisplayHeader(Url string, Method string) {
	ServerHeader, Status, Title, err := HttpGetServerHeader(Url, true, Method)
	if err != nil {
		color.HiRed("Error: %s\n", err)
	} else {
		color.Cyan("Url: %s\tMethod: %s\n", Url, Method)
		color.Cyan("Server: %s\tStatus: %s\tTitle: %s\n", ServerHeader, Status, Title)
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
	req.Header.Set("User-Agent", "Mozilla/5.0 (Macintosh; Intel Mac OS X 100) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/1.0.5005.61 Safari/537.36")
	resp, err := Client.Do(req)

	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	bodyBytes, _ := ioutil.ReadAll(resp.Body)
	ico := Mmh3Hash32(StandBase64(bodyBytes))
	color.Red("[*] icon_hash `%s`", ico)
	var icon_hash_map map[string]interface{}
	json.Unmarshal([]byte(icon_json), &icon_hash_map)
	tmp := icon_hash_map[ico]
	if tmp != nil {
		color.Red("[*] icon_hash `%s`", tmp)
	}
	return "", nil
}

func PrintFinger(Info common.HostInfo) {
	InitHttp()
	color.HiRed("Your URL: %s\n", Info.Url)
	Host, _ := url.Parse(Info.Url)
	RootPath := Host.Scheme + "://" + Host.Hostname()
	if Host.Port() != "" {
		RootPath = RootPath + ":" + Host.Port()
	}
	// 首页
	FirstUrl := RootPath + Host.Path
	DisplayHeader(FirstUrl, http.MethodGet)

	// 构造404
	SecondUrl := RootPath + "/xxxxxx"
	DisplayHeader(SecondUrl, http.MethodGet)

	// 构造POST
	ThirdUrl := RootPath
	DisplayHeader(ThirdUrl, http.MethodPost)

	IconUrl := RootPath + "/favicon.ico"
	IconDetect(IconUrl)

	// 爬虫递归爬
	s1 := mapset.NewSet()
	// fmt.Print(s1)
	Spider(RootPath, Info.Url, Info.Depth, s1)
}
