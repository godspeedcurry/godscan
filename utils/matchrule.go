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
)

func iskeyword(str string, keyword []string) bool {
	var x bool
	x = true
	for _, k := range keyword {
		if strings.Contains(str, k) {
			x = x && true
		} else {
			x = x && false
		}
	}
	return x
}

func isregular(str string, keyword []string) bool {
	var x bool
	x = true
	for _, k := range keyword {
		re := regexp.MustCompile(k)
		if re.Match([]byte(str)) {
			x = x && true
		} else {
			x = x && false
		}
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

func fingerScan(url string) string {
	InitHttp()

	req, _ := http.NewRequest(http.MethodGet, url, nil)
	req.Header.Set("User-Agent", "Mozilla/5.0 (Macintosh; Intel Mac OS X 100) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/1.0.5005.61 Safari/537.36")
	req.Header.Set("Cookie", "rememberMe=me")

	resp, err := Client.Do(req)

	if err != nil {
		fmt.Println(err)
		return "no finger!!"
	}
	headers := MapToJson(resp.Header)

	var config Packjson

	err = json.Unmarshal([]byte(eholeJson), &config)
	if err != nil {
		fmt.Println(err)
		return "no finger!!"
	}
	var cms []string
	bodyBytes, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		fmt.Println(err)
		return "no finger!!"
	}
	body := string(bodyBytes)

	doc, err := goquery.NewDocumentFromReader(resp.Body)
	if err != nil {
		return "no finger!!"
	}

	// 查找标题元素并获取内容
	title := doc.Find("title").Text()

	for _, finp := range config.Fingerprint {

		if finp.Location == "body" {
			if finp.Method == "keyword" {
				if iskeyword(body, finp.Keyword) {
					cms = append(cms, finp.Cms)
				}
			}

			if finp.Method == "regular" {
				if isregular(body, finp.Keyword) {
					cms = append(cms, finp.Cms)
				}
			}
		}
		if finp.Location == "header" {
			if finp.Method == "keyword" {
				if iskeyword(headers, finp.Keyword) {
					cms = append(cms, finp.Cms)
				}
			}
			if finp.Method == "regular" {
				if isregular(headers, finp.Keyword) {
					cms = append(cms, finp.Cms)
				}
			}
		}
		if finp.Location == "title" {
			if finp.Method == "keyword" {
				if iskeyword(title, finp.Keyword) {
					cms = append(cms, finp.Cms)
				}
			}
			if finp.Method == "regular" {
				if isregular(title, finp.Keyword) {
					cms = append(cms, finp.Cms)
				}
			}
		}
	}
	if len(cms) != 0 {
		return strings.Join(cms, ",")
	}
	return "no finger!!"
}
