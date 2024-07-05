package utils

import (
	"bytes"
	_ "embed"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"

	"github.com/Knetic/govaluate"
	regexp2 "github.com/dlclark/regexp2"

	"regexp"
	"strings"

	"github.com/PuerkitoBio/goquery"
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

// 递归解析表达式，考虑括号和逻辑优先级
func preprocessAndEvaluate(input string, context map[string]string) (bool, error) {
	// 使用 regexp2 包来替换原来的 regexp
	var re = regexp2.MustCompile(`\(([^()]*)\)|((\w+)\s*=\s*"((?:[^"]|"(?! && | \|\| |$))*)")`, regexp2.None)
	// 不断替换直到无括号为止
	for {
		matches, _ := re.FindStringMatch(input)
		if matches == nil {
			break
		}

		allMatches := []*regexp2.Match{}
		for matches != nil {
			allMatches = append(allMatches, matches)
			matches, _ = re.FindNextMatch(matches)
		}
		for _, match := range allMatches {
			if match.Groups()[1].String() != "" { // 匹配到括号内的表达式
				result, err := preprocessAndEvaluate(match.Groups()[1].String(), context)
				if err != nil {
					return false, err
				}
				resultStr := "false"
				if result {
					resultStr = "true"
				}
				input = strings.Replace(input, fmt.Sprintf("(%s)", match.Groups()[1].String()), resultStr, 1)
			} else if match.Groups()[2].String() != "" { // 匹配到键值对表达式
				key, value := match.Groups()[3].String(), match.Groups()[4].String()
				if strings.Contains(context[key], value) {
					input = strings.Replace(input, match.Groups()[2].String(), "true", 1)
				} else {
					input = strings.Replace(input, match.Groups()[2].String(), "false", 1)
				}
			}
		}
	}

	input = strings.TrimSpace(input)
	if input == "" {
		return false, nil
	}

	// 使用 govaluate 解析最终表达式
	expr, err := govaluate.NewEvaluableExpression(input)
	if err != nil {
		Error("%s %s", err, input)
		return false, err
	}
	result, err := expr.Evaluate(nil) // nil因为我们已经将所有东西预处理为true/false
	if err != nil {
		Error("%s %s", err, input)
		return false, err
	}

	return result.(bool), nil
}

func FingerScan(url string, method string, followRedirect bool) (string, string, string, string, string, []byte, int) {
	if !isValidUrl(url) {
		return common.NoFinger, "", "", "", "", nil, -1
	}
	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		Fatal("%s %s xxx", url, err)
		return common.NoFinger, "", "", "", "", nil, -1
	}
	req.Header.Set("User-Agent", viper.GetString("DefaultUA"))
	req.Header.Set("Cookie", "rememberMe=me")
	if !followRedirect {
		Client.CheckRedirect = func(req *http.Request, via []*http.Request) error {
			return http.ErrUseLastResponse
		}
	}

	resp, err := Client.Do(req)
	if err != nil {
		Fatal("%s %s create request failed", url, err)
		return common.NoFinger, "", "", "", "", nil, -1
	}
	defer resp.Body.Close()
	ServerValue := resp.Header["Server"]
	retServerValue := ""
	if len(ServerValue) != 0 {
		retServerValue = ServerValue[0]
	}

	if resp.StatusCode == 301 || resp.StatusCode == 302 {
		return common.NoFinger, retServerValue, "", resp.Header.Get("Content-Type"), resp.Header.Get("Location"), nil, resp.StatusCode
	}
	headers := MapToJson(resp.Header)

	var config Packjson

	err = json.Unmarshal([]byte(eholeJson), &config)
	if err != nil {
		Fatal("%s %s unmarshal failed", url, err)
		return common.NoFinger, "", "", "", "", nil, -1
	}
	ContentLengthStr := resp.Header.Get("Content-Length")
	ServerContentType := resp.Header.Get("Content-Type")
	ContentLength, err := strconv.Atoi(ContentLengthStr)
	if err != nil {
		Fatal("%s %s %s", url, err, ContentLengthStr)
		return common.NoFinger, "", "", "", "", nil, -1
	}
	if ContentLength > 1024*1024*10 { // 10M
		return "Large Data", retServerValue, "Large Data size = [" + ContentLengthStr + "]", resp.Header.Get("Content-Type"), resp.Header.Get("Location"), nil, resp.StatusCode
	}
	var cms []string
	bodyBytes, _ := io.ReadAll(resp.Body)
	_, DetermineContentType, _ := charset.DetermineEncoding(bodyBytes, ServerContentType)
	reader, err := charset.NewReader(bytes.NewBuffer(bodyBytes), DetermineContentType)
	if err != nil {
		Fatal("%s %s %s", url, err, DetermineContentType)
		return common.NoFinger, "", "", "", "", nil, -1
	}
	doc, err := goquery.NewDocumentFromReader(reader)

	if err != nil {
		Fatal("%s %s", url, err)
		return common.NoFinger, "", "", "", "", nil, -1
	}

	// 查找标题元素并获取内容
	title := strings.TrimSpace(doc.Find("title").Text())

	body := string(bodyBytes)
	context := map[string]string{
		"title":  title,
		"body":   body,
		"server": headers,
		"header": headers,
	}
	for _, fp := range config.Fingerprint {
		matcher, found := methodMatchers[fp.Method]
		if found {
			locator := chooseLocator(headers, body, title, fp)
			if matcher(locator, fp.Keyword) {
				cms = append(cms, fp.Cms)
			}
		} else {
			// 逻辑表达式 location字段不重要
			res, err := preprocessAndEvaluate(fp.Keyword[0], context)
			if err != nil {
				Error("%s %s", err, fp.Keyword[0])
				continue
			}
			if res {
				cms = append(cms, fp.Cms)
			}
		}
	}
	finger := common.NoFinger
	cms = RemoveDuplicatesString(cms)
	if len(cms) != 0 {
		finger = strings.Join(cms, ",")
	}

	return finger, retServerValue, title, resp.Header.Get("Content-Type"), resp.Header.Get("Location"), []byte(bodyBytes), resp.StatusCode
}
