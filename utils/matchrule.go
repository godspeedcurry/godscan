package utils

import (
	_ "embed"
	"encoding/json"
	"io"
	"net/http"
	"regexp"
	"strings"

	"github.com/PuerkitoBio/goquery"
	"github.com/godspeedcurry/godscan/common"
	"github.com/spf13/viper"
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

func FingerScan(url string) (string, string, []byte, int) {
	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		Fatal("%s", err)
		return common.NoFinger, "", nil, -1
	}
	req.Header.Set("User-Agent", viper.GetString("DefaultUA"))
	req.Header.Set("Cookie", "rememberMe=me")
	resp, err := Client.Do(req)
	if err != nil {
		Fatal("%s", err)
		return common.NoFinger, "", nil, -1
	}
	defer resp.Body.Close()
	headers := MapToJson(resp.Header)

	var config Packjson

	err = json.Unmarshal([]byte(eholeJson), &config)
	if err != nil {
		Fatal("%s", err)
		return common.NoFinger, "", nil, -1
	}
	var cms []string
	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		Fatal("%s", err)
		return common.NoFinger, "", nil, -1
	}
	body := string(bodyBytes)

	doc, err := goquery.NewDocumentFromReader(resp.Body)
	if err != nil {
		return common.NoFinger, "", nil, -1
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
		return strings.Join(cms, ","), "", nil, -1
	}
	return common.NoFinger, resp.Header.Get("Content-Type"), bodyBytes, resp.StatusCode
}
