package utils

import (
	"bufio"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"os"
	"regexp"
	"sort"
	"strconv"
	"strings"

	"github.com/PuerkitoBio/goquery"
	mapset "github.com/deckarep/golang-set"
	"github.com/fatih/color"
	"github.com/scylladb/termtables"
)

func mysort(mymap map[string]int) PairList {
	pl := make(PairList, len(mymap))
	i := 0
	for k, v := range mymap {
		pl[i] = Pair{k, v}
		i++
	}
	sort.Sort(sort.Reverse(pl))
	return pl
}

type Pair struct {
	Key   string
	Value int
}

type PairList []Pair

func (p PairList) Len() int           { return len(p) }
func (p PairList) Less(i, j int) bool { return p[i].Value < p[j].Value }
func (p PairList) Swap(i, j int)      { p[i], p[j] = p[j], p[i] }

func HttpGetServerHeader(Url string, NeedTitle bool) (string, string, error) {
	req, _ := http.NewRequest(http.MethodGet, Url, nil)
	req.Header.Set("User-Agent", "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/102.0.5005.61 Safari/537.36")
	resp, err := Client.Do(req)
	if err != nil {
		return "", "", err
	}
	defer resp.Body.Close()
	doc, err := goquery.NewDocumentFromReader(resp.Body)
	title := doc.Find("title").Text()
	if err != nil {
		log.Fatal(err)
	}
	ServerValue := resp.Header["Server"]
	if len(ServerValue) != 0 {
		return ServerValue[0], title, nil
	}
	return "", "", nil
}

func Normalize(Path string, RootPath string) string {
	if strings.Contains(Path, "javascript:") {
		return ""
	} else if strings.HasPrefix(Path, "http://") {
		return Path
	} else if strings.HasPrefix(Path, "https://") {
		return Path
	} else if strings.HasPrefix(Path, "./") {
		return RootPath + Path[1:]
	} else if strings.HasPrefix(Path, "/") {
		return RootPath + Path
	} else {
		return RootPath + "/" + Path
	}
}

func StringListToInterfaceList(tmpList []string) []interface{} {
	vals := make([]interface{}, len(tmpList))
	for i, v := range tmpList {
		vals[i] = v
	}
	return vals
}
func FindKeyWord(data string) {
	fi, err := os.Open("utils/finger.txt")

	m := make(map[string]int)
	if err != nil {
		fmt.Printf("Error: %s\n", err)
		return
	}
	defer fi.Close()

	br := bufio.NewReader(fi)

	for {
		a, _, c := br.ReadLine()
		if c == io.EOF {
			break
		}
		x := string(a)
		if strings.Contains(data, x) {
			cnt := strings.Count(data, x)
			m[x] = cnt
		}
	}
	y := mysort(m)
	table := termtables.CreateTable()
	table.AddHeaders("Key1", "Value1", "Key2", "Value2", "Key3", "Value3", "Key4", "Value4", "Key5", "Value5")
	tmpList := []string{}
	cnt := 0
	for _, tmp := range y {
		tmpList = append(tmpList, tmp.Key)
		tmpList = append(tmpList, strconv.Itoa(tmp.Value))
		cnt++
		if cnt%5 == 0 {
			table.AddRow(StringListToInterfaceList(tmpList)...)
			tmpList = []string{}
		}
	}
	if cnt%5 != 0 {
		for i := 0; i <= 5-cnt%5; i++ {
			tmpList = append(tmpList, "None")
			tmpList = append(tmpList, "None")
		}
		table.AddRow(StringListToInterfaceList(tmpList)...)
	}
	color.Cyan("%s\n", table.Render())
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
	req.Header.Set("User-Agent", "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/102.0.5005.61 Safari/537.36")
	resp, err := Client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	doc, _ := goquery.NewDocumentFromReader(resp.Body)
	FindKeyWord(doc.Text())
	//正则提取注释
	reg := regexp.MustCompile("/\\*[\u0000-\uffff]{1,300}?\\*/")

	result := reg.FindAllString(strings.ReplaceAll(doc.Text(), "\t", ""), -1)
	fmt.Println(result)

	// a标签
	doc.Find("a").Each(func(i int, a *goquery.Selection) {
		href, _ := a.Attr("href")
		normalizeUrl := Normalize(href, RootPath)
		if normalizeUrl != "" && !s1.Contains(normalizeUrl) {
			Spider(RootPath, normalizeUrl, depth-1, s1)
		}
	})
	// script 标签
	doc.Find("script").Each(func(i int, a *goquery.Selection) {
		src, _ := a.Attr("src")
		normalizeUrl := Normalize(src, RootPath)
		if normalizeUrl != "" && !s1.Contains(normalizeUrl) {
			Spider(RootPath, normalizeUrl, depth-1, s1)
		}
	})
	return "", nil
}

func DisplayHeader(Url string) {
	ServerHeader, Title, err := HttpGetServerHeader(Url, true)
	if err != nil {
		color.HiRed("Error: %s\n", err)
	} else {
		color.Cyan("Server: %s\n", ServerHeader)
		color.Cyan("Title: %s\n", Title)
	}
}

func PrintFinger(Url string) {
	InitHttp()
	color.HiRed("Your URL: %s\n", Url)
	Host, _ := url.Parse(Url)

	// 首页
	FirstUrl := Host.Scheme + "://" + Host.Hostname()
	DisplayHeader(FirstUrl)

	// 构造404
	SecondUrl := Host.Scheme + "://" + Host.Hostname() + "/xxxxxx"
	DisplayHeader(SecondUrl)

	// 爬虫递归爬
	s1 := mapset.NewSet()
	Spider(Host.Scheme+"://"+Host.Hostname(), Url, 10, s1)
}
