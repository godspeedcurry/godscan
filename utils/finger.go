package utils

import (
	"fmt"
	"log"
	"net/http"
	"net/url"
	"regexp"
	"strconv"
	"strings"

	"github.com/PuerkitoBio/goquery"
	mapset "github.com/deckarep/golang-set"
	"github.com/fatih/color"
	"github.com/gobuffalo/packr"
	"github.com/scylladb/termtables"
)

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
	if len(ServerValue) != 0 {
		return ServerValue[0], Status, title, nil
	}
	return "", "", "", nil
}

func FindKeyWord(data string) {
	m := make(map[string]int)
	box := packr.NewBox(".")
	fingers, err := box.FindString("finger.txt")
	if err != nil {
		fmt.Println(err)
		return
	}
	var fingerList = strings.Split(fingers, "\r\n")
	for _, finger := range fingerList {
		if strings.Contains(data, finger) {
			cnt := strings.Count(data, finger)
			m[finger] = cnt
		}
	}

	mSorted := mysort(m)
	table := termtables.CreateTable()
	table.AddHeaders("Key1", "Value1", "Key2", "Value2", "Key3", "Value3", "Key4", "Value4", "Key5", "Value5")
	tmpList := []string{}
	cnt := 0
	maxColumn := 10
	for _, tmp := range mSorted {
		tmpList = append(tmpList, tmp.Key)
		tmpList = append(tmpList, strconv.Itoa(tmp.Value))
		cnt++
		if cnt%(maxColumn/2) == 0 {
			table.AddRow(StringListToInterfaceList(tmpList[:maxColumn])...)
			tmpList = []string{}
		}
	}
	if cnt%5 != 0 {
		tmpList = append(tmpList, make([]string, maxColumn)...)
		table.AddRow(StringListToInterfaceList(tmpList[:maxColumn])...)
	}
	color.Cyan("%s\n", table.Render())
}

func IsVuePath(Path string) bool {
	reg := regexp.MustCompile(`app\.[0-9a-z]+\.js`)
	res := reg.FindAllString(Path, -1)
	return len(res) > 0
}

func Spider(RootPath string, Url string, depth int, s1 mapset.Set) (string, error) {
	if !strings.Contains(Url, RootPath) {
		fmt.Printf("======Depth %d, target %s =====\n", depth, Url)
		s1.Add(Url)
		return "", nil
	} else if depth == 0 || strings.Contains(Url, ".min.js") || strings.Contains(Url, ".ico") {
		return "", nil
	}
	fmt.Printf("======Depth %d, target %s =====\n", depth, Url)
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
	FindKeyWord(doc.Text())

	//正则提取注释
	AnnotationReg := regexp.MustCompile("/\\*[\u0000-\uffff]{1,300}?\\*/")

	AnnotationResult := AnnotationReg.FindAllString(strings.ReplaceAll(doc.Text(), "\t", ""), -1)
	if len(AnnotationResult) > 0 {
		fmt.Println("[*] 注释部分")
		fmt.Println(AnnotationResult)
	}

	//正则提取版本
	VersionReg := regexp.MustCompile(`(?i)(version|ver|v|版本)[ =:]{0,2}(\d+)(\.[0-9a-z]+)*`)

	VersionResult := VersionReg.FindAllString(strings.ReplaceAll(doc.Text(), "\t", ""), -1)
	if len(VersionResult) > 0 {
		fmt.Println("[*] 版本识别")
		res, _ := removeDuplicateElement(VersionResult)
		fmt.Println(strings.Join(res.([]string), "\n"))
	}

	// 如果是vue.js app.xxxxxxxx.js
	if IsVuePath(Url) {
		fmt.Println("[*] Api Path")
		ApiReg := regexp.MustCompile(`path:"(?P<path>.*?)"`)
		ApiResult := ApiReg.FindAllStringSubmatch(strings.ReplaceAll(doc.Text(), "\t", ""), -1)

		if len(ApiResult) > 0 {
			for _, tmp := range ApiResult {
				fmt.Println(RootPath + tmp[1])
			}
		}

	}

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

func PrintFinger(Url string) {
	InitHttp()
	color.HiRed("Your URL: %s\n", Url)
	Host, _ := url.Parse(Url)
	RootPath := Host.Scheme + "://" + Host.Hostname()
	if Host.Port() != "" {
		RootPath = RootPath + ":" + Host.Port()
	}
	// 首页
	FirstUrl := RootPath
	DisplayHeader(FirstUrl, http.MethodGet)

	// 构造404
	SecondUrl := RootPath + "/xxxxxx"
	DisplayHeader(SecondUrl, http.MethodGet)

	// 构造POST
	ThirdUrl := RootPath
	DisplayHeader(ThirdUrl, http.MethodPost)

	// 爬虫递归爬
	s1 := mapset.NewSet()
	// fmt.Print(s1)
	Spider(RootPath, Url, 10, s1)
}
