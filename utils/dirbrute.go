package utils

import (
	"net/url"
	"path"
	"strconv"
	"strings"

	"github.com/fatih/color"
)

var fingerHashMap = make(map[uint64]bool)

func formatUrl(raw string) string {
	if !strings.HasPrefix(raw, "http") {
		raw = "http://" + raw
	}
	return strings.TrimSpace(raw)
}

func CheckFinger(finger string, title string, url string, contentType string, respBody []byte, statusCode int) []string {
	if len(title) > 50 {
		title = title[:50] + "..."
	}
	hash := SimHash(respBody)
	if !fingerHashMap[hash] {
		fingerHashMap[hash] = true
		return []string{url, title, finger, contentType, strconv.Itoa(statusCode), strconv.Itoa(len(respBody))}
	}
	return []string{}
}

func DirBrute(baseUrl string, dir string) []string {
	result := []string{}
	baseURL, err := url.Parse(formatUrl(baseUrl))
	if err != nil {
		Error("%s", err)
		return []string{}
	}
	fullURL := baseURL.ResolveReference(&url.URL{Path: path.Join(baseURL.Path, dir)})
	finger, _, title, contentType, respBody, statusCode := FingerScan(fullURL.String())
	if statusCode == 200 || statusCode == 500 {
		result = CheckFinger(finger, title, fullURL.String(), contentType, respBody, statusCode)
	}
	if len(result) > 0 {
		WriteToCsv("dirbrute.csv", result)
	}

	if len(result) > 0 {
		result[2] = color.GreenString(result[2])
	}
	return result
}
