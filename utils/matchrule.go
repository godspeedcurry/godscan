package utils

import (
	"bytes"
	_ "embed"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"

	"github.com/Knetic/govaluate"
	regexp2 "github.com/dlclark/regexp2"
	"github.com/spf13/viper"

	"regexp"
	"strings"

	"github.com/PuerkitoBio/goquery"
	"github.com/godspeedcurry/godscan/common"
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

func logRequestError(targetURL string, err error) {
	host := targetURL
	if u, parseErr := url.Parse(targetURL); parseErr == nil && u.Hostname() != "" {
		host = u.Hostname()
	}
	recordHostError(host, err.Error())
	if _, loaded := hostErrorOnce.LoadOrStore(host, struct{}{}); !loaded {
		Warning("Skip %s: %v", targetURL, err)
		return
	}
	Debug("Skip %s: %v", targetURL, err)
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

type FingerResult struct {
	Finger      string
	Server      string
	Title       string
	ContentType string
	Location    string
	HeadersJSON string
	Body        []byte
	Status      int
	Err         error
}

func FingerScan(url string, method string, followRedirect bool) FingerResult {
	if !isValidUrl(url) {
		return FingerResult{Finger: common.NoFinger, Status: -1, Err: fmt.Errorf("invalid url")}
	}
	resp, serverHeader, err := doHTTPRequest(url, method, followRedirect)
	if err != nil {
		return FingerResult{Finger: common.NoFinger, Status: -1, Err: err}
	}
	defer resp.Body.Close()

	if !followRedirect && isRedirect(resp.StatusCode) {
		return FingerResult{
			Finger:      common.NoFinger,
			Server:      serverHeader,
			Title:       "",
			ContentType: resp.Header.Get("Content-Type"),
			Location:    resp.Header.Get("Location"),
			HeadersJSON: MapToJson(resp.Header),
			Status:      resp.StatusCode,
		}
	}

	headersJSON := MapToJson(resp.Header)
	config, err := loadFingerConfig(url)
	if err != nil {
		return FingerResult{Finger: common.NoFinger, HeadersJSON: headersJSON, Status: -1, Err: err}
	}

	bodyBytes, contentLength := readResponseBody(resp)
	if contentLength > 10*1024*1024 {
		return FingerResult{
			Finger:      "Large Data",
			Server:      serverHeader,
			Title:       fmt.Sprintf("Large Data size = [%d]", contentLength),
			ContentType: resp.Header.Get("Content-Type"),
			Location:    resp.Header.Get("Location"),
			HeadersJSON: headersJSON,
			Status:      resp.StatusCode,
		}
	}

	doc, title, err := buildDocument(bodyBytes, resp.Header.Get("Content-Type"))
	if err != nil {
		return FingerResult{Finger: common.NoFinger, HeadersJSON: headersJSON, Status: -1, Err: err}
	}

	finger := matchFingerprints(doc, string(bodyBytes), title, headersJSON, config)
	return FingerResult{
		Finger:      finger,
		Server:      serverHeader,
		Title:       title,
		ContentType: resp.Header.Get("Content-Type"),
		Location:    resp.Header.Get("Location"),
		HeadersJSON: headersJSON,
		Body:        bodyBytes,
		Status:      resp.StatusCode,
	}
}

func doHTTPRequest(url, method string, followRedirect bool) (*http.Response, string, error) {
	req, err := http.NewRequest(method, url, nil)
	if err != nil {
		logRequestError(url, err)
		return nil, "", err
	}
	req.Header.Set("Cookie", "rememberMe=me")
	SetHeaders(req)
	var client *http.Client
	if followRedirect {
		client = Client
		if client == nil {
			client = http.DefaultClient
		}
	} else {
		client = enforceNoRedirectClient(ClientNoRedirect)
	}
	resp, err := client.Do(req)
	if err != nil {
		logRequestError(url, err)
		return nil, "", err
	}
	serverHeader := ""
	if sv := resp.Header["Server"]; len(sv) > 0 {
		serverHeader = sv[0]
	}
	return resp, serverHeader, nil
}

func enforceNoRedirectClient(base *http.Client) *http.Client {
	if base == nil {
		return &http.Client{
			CheckRedirect: func(req *http.Request, via []*http.Request) error { return http.ErrUseLastResponse },
		}
	}
	clone := *base
	clone.CheckRedirect = func(req *http.Request, via []*http.Request) error { return http.ErrUseLastResponse }
	return &clone
}

func isRedirect(status int) bool {
	return status == 301 || status == 302 || status == 303 || status == 307 || status == 308
}

func loadFingerConfig(url string) (Packjson, error) {
	var config Packjson
	if err := json.Unmarshal([]byte(eholeJson), &config); err != nil {
		Fatal("%s %s unmarshal failed", url, err)
		return config, err
	}
	return config, nil
}

func readResponseBody(resp *http.Response) ([]byte, int) {
	maxBody := viper.GetInt("max-body-bytes")
	if maxBody <= 0 || maxBody > 4*1024*1024 {
		maxBody = 4 * 1024 * 1024
	}
	bodyBytes, _ := io.ReadAll(io.LimitReader(resp.Body, int64(maxBody)))
	contentLength := len(bodyBytes)
	if cl := resp.Header.Get("Content-Length"); cl != "" {
		if v, e := strconv.Atoi(cl); e == nil {
			contentLength = v
		}
	}
	return bodyBytes, contentLength
}

func buildDocument(body []byte, serverContentType string) (*goquery.Document, string, error) {
	if len(body) == 0 {
		doc, _ := goquery.NewDocumentFromReader(strings.NewReader("<html></html>"))
		return doc, "", nil
	}
	_, determineContentType, _ := charset.DetermineEncoding(body, serverContentType)
	reader, err := charset.NewReader(bytes.NewBuffer(body), determineContentType)
	if err != nil {
		return nil, "", fmt.Errorf("%v (%s)", err, determineContentType)
	}
	doc, err := goquery.NewDocumentFromReader(reader)
	if err != nil {
		return nil, "", err
	}
	title := strings.TrimSpace(doc.Find("title").Text())
	return doc, title, nil
}

func matchFingerprints(doc *goquery.Document, body string, title string, headers string, config Packjson) string {
	context := map[string]string{
		"title":  title,
		"body":   body,
		"server": headers,
		"header": headers,
	}
	var cms []string
	for _, fp := range config.Fingerprint {
		matcher, found := methodMatchers[fp.Method]
		if found {
			locator := chooseLocator(headers, body, title, fp)
			if matcher(locator, fp.Keyword) {
				cms = append(cms, fp.Cms)
			}
			continue
		}
		res, err := preprocessAndEvaluate(fp.Keyword[0], context)
		if err != nil {
			Error("%s %s", err, fp.Keyword[0])
			continue
		}
		if res {
			cms = append(cms, fp.Cms)
		}
	}
	cms = RemoveDuplicatesString(cms)
	if len(cms) == 0 {
		return common.NoFinger
	}
	return strings.Join(cms, ",")
}
