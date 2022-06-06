package utils

import (
	"fmt"
	"log"
	"net/http"
	"net/url"
	"strings"

	"github.com/PuerkitoBio/goquery"
	mapset "github.com/deckarep/golang-set"
	"github.com/fatih/color"
)

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

func Normalize(Path string, Url string) string {
	u, _ := url.Parse(Url)
	normalizeUrl := u.Scheme + "://" + u.Hostname()

	if strings.Contains(Path, "javascript:") {
		return ""
	} else if strings.HasPrefix(Path, "http://") {
		return Path
	} else if strings.HasPrefix(Path, "https://") {
		return Path
	} else if strings.HasPrefix(Path, "./") {
		return normalizeUrl + Path[1:]
	} else if strings.HasPrefix(Path, "/") {
		return normalizeUrl + Path
	} else {
		return normalizeUrl + "/" + Path
	}
}

func Spider(Host string, Url string, depth int, s1 mapset.Set) (string, error) {
	if !strings.Contains(Url, Host) {
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
	// a标签
	doc.Find("a").Each(func(i int, a *goquery.Selection) {
		href, _ := a.Attr("href")
		normalizeUrl := Normalize(href, Url)
		if normalizeUrl != "" && !s1.Contains(normalizeUrl) {
			Spider(Host, normalizeUrl, depth-1, s1)
		}
	})
	// script 标签
	doc.Find("script").Each(func(i int, a *goquery.Selection) {
		src, _ := a.Attr("src")
		normalizeUrl := Normalize(src, Url)
		if normalizeUrl != "" && !s1.Contains(normalizeUrl) {
			Spider(Host, normalizeUrl, depth-1, s1)
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

	s1 := mapset.NewSet()
	Spider(Host.Hostname(), Url, 2, s1)

}
