package utils

import (
	"fmt"
	"net/url"
	"path"
	"strings"
)

var fingerHashMap = make(map[uint64]bool)
var result = []string{}

func formatUrl(raw string) string {
	if !strings.HasPrefix(raw, "http") {
		raw = "http://" + raw
	}
	return strings.TrimSpace(raw)
}

func CheckFinger(finger string, url string, contentType string, respBody []byte, statusCode int) string {
	hash := SimHash(respBody)
	if !fingerHashMap[hash] {
		fingerHashMap[hash] = true
		return fmt.Sprintf("[%d] %d {%s} {%s} %s", statusCode, len(respBody), finger, contentType, url)
	}
	return ""
}

func DirBrute(baseUrl string, dirlist []string) []string {
	result := []string{}
	baseURL, err := url.Parse(formatUrl(baseUrl))
	if err != nil {
		Error("%s", err)
		return []string{}
	}
	for _, _path := range dirlist {
		fullURL := baseURL.ResolveReference(&url.URL{Path: path.Join(baseURL.Path, _path)})
		finger, contentType, respBody, statusCode := FingerScan(fullURL.String())
		if statusCode == 200 || statusCode == 500 {
			ret := CheckFinger(finger, fullURL.String(), contentType, respBody, statusCode)
			if ret != "" {
				Info(fullURL.String() + " " + ret)
				result = append(result, ret)
			}
		}
	}
	return result
}
