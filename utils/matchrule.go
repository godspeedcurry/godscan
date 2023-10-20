package utils

import (
	_ "embed"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"regexp"
	"strings"

	"github.com/PuerkitoBio/goquery"
	"github.com/godspeedcurry/godscan/common"
)

func iskeyword(str string, keyword []string) bool {
	var x bool = true
	for _, k := range keyword {
		x = x && strings.Contains(str, k)
	}
	return x
}

func isregular(str string, keyword []string) bool {
	var x bool
	x = true
	for _, k := range keyword {
		re := regexp.MustCompile(k)
		x = x && re.Match([]byte(str))
	}
	return x
}

func MapToJson(param map[string][]string) string {
	dataType, _ := json.Marshal(param)
	dataString := string(dataType)
	return dataString
}

type Packjson struct {
	Fingerprint []Fingerprint
}

type Fingerprint struct {
	Cms      string
	Method   string
	Location string
	Keyword  []string
}

//go:embed ehole.json
var eholeJson string

type methodMatcher func(content string, keyword []string) bool

var methodMatchers = map[string]methodMatcher{
	"keyword": iskeyword,
	"regular": isregular,
}

func chooseLocator(headers string, body string, title string, fp Fingerprint) string {
	if fp.Location == "header" {
		return headers
	} else if fp.Location == "body" {
		return body
	} else if fp.Location == "title" {
		return title
	}
	return ""
}

func fingerScan(url string) string {
	InitHttp()
	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return common.NoFinger
	}
	req.Header.Set("User-Agent", common.DEFAULT_UA)
	req.Header.Set("Cookie", "rememberMe=me")

	resp, err := Client.Do(req)

	if err != nil {
		return common.NoFinger
	}
	headers := MapToJson(resp.Header)

	var config Packjson

	err = json.Unmarshal([]byte(eholeJson), &config)
	if err != nil {
		fmt.Println(err)
		return common.NoFinger
	}
	var cms []string
	bodyBytes, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		fmt.Println(err)
		return common.NoFinger
	}
	body := string(bodyBytes)

	doc, err := goquery.NewDocumentFromReader(resp.Body)
	if err != nil {
		return common.NoFinger
	}

	// 查找标题元素并获取内容
	title := doc.Find("title").Text()

	for _, fp := range config.Fingerprint {
		matcher, found := methodMatchers[fp.Method]
		if found {
			locator := chooseLocator(headers, body, title, fp)
			if matcher(locator, fp.Keyword) {
				cms = append(cms, fp.Cms)
			}
		}

	}
	if len(cms) != 0 {
		return strings.Join(cms, ",")
	}
	return common.NoFinger
}
