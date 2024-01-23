package utils

import (
	"bytes"
	_ "embed"
	"encoding/json"
	"io"
	"net/http"
	"regexp"
	"strings"

	"github.com/PuerkitoBio/goquery"
	"github.com/fatih/color"
	"github.com/godspeedcurry/godscan/common"
	"github.com/spf13/viper"
	"golang.org/x/net/html/charset"
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

func FingerScan(url string) (string, string, string, string, []byte, int) {
	if !isValidUrl(url) {
		return common.NoFinger, "", "", "", nil, -1
	}
	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		Fatal("%s", err)
		return common.NoFinger, "", "", "", nil, -1
	}
	req.Header.Set("User-Agent", viper.GetString("DefaultUA"))
	req.Header.Set("Cookie", "rememberMe=me")
	resp, err := Client.Do(req)
	if err != nil {
		Fatal("%s", err)
		return common.NoFinger, "", "", "", nil, -1
	}
	defer resp.Body.Close()
	headers := MapToJson(resp.Header)

	var config Packjson

	err = json.Unmarshal([]byte(eholeJson), &config)
	if err != nil {
		Fatal("%s", err)
		return common.NoFinger, "", "", "", nil, -1
	}
	var cms []string
	bodyBytes, _ := io.ReadAll(resp.Body)
	_, contentType, _ := charset.DetermineEncoding(bodyBytes, resp.Header.Get("Content-Type"))
	reader, _ := charset.NewReader(bytes.NewBuffer(bodyBytes), contentType)
	doc, err := goquery.NewDocumentFromReader(reader)

	if err != nil {
		Fatal("%s", err)
		return common.NoFinger, "", "", "", nil, -1
	}

	// 查找标题元素并获取内容
	title := strings.TrimSpace(doc.Find("title").Text())

	for _, fp := range config.Fingerprint {
		matcher, found := methodMatchers[fp.Method]
		if found {
			locator := chooseLocator(headers, string(bodyBytes), title, fp)
			if matcher(locator, fp.Keyword) {
				cms = append(cms, fp.Cms)
			}
		}
	}
	finger := common.NoFinger

	if len(cms) != 0 {
		finger = color.GreenString(strings.Join(cms, ","))
	}
	ServerValue := resp.Header["Server"]
	retServerValue := ""
	if len(ServerValue) != 0 {
		retServerValue = ServerValue[0]
	}
	return finger, retServerValue, title, resp.Header.Get("Content-Type"), bodyBytes, resp.StatusCode
}
