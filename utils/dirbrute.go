package utils

import (
	"encoding/json"
	"fmt"
	"net"
	"net/url"
	"path"
	"strings"

	"github.com/godspeedcurry/godscan/common"
)

var fingerHashMap = make(map[uint64]bool)
var result = []string{}

func formatUrl(raw string) string {
	if !strings.HasPrefix(raw, "http") {
		raw = "http://" + raw
	}
	return strings.TrimSpace(raw)
}

func JsonToMap(data string) string {
	headers := make(map[string]interface{})
	_ = json.Unmarshal([]byte(data), &headers)
	// fmt.Println(data)
	if headers["content-type"] != nil {
		return headers["content-type"].(string)
	}
	return ""
}
func CheckFinger(finger string, url string, headers string, respBody []byte, statusCode int) string {
	hash := SimHash(respBody)
	header := JsonToMap(headers)
	if !fingerHashMap[hash] {
		fingerHashMap[hash] = true
		return fmt.Sprintf("[%d] %d {%s} %s {%s} [%s]", statusCode, len(respBody), finger, url, header, string(respBody[:50]))
	}
	return ""
}

func DirBrute(baseUrl string) []string {
	result := []string{}
	baseURL, err := url.Parse(formatUrl(baseUrl))
	if err != nil {
		fmt.Println(err)
		return []string{}
	}
	tempDirList := common.DirList

	ip := net.ParseIP(baseURL.Hostname())
	if ip == nil {
		// is a domain
		parts := strings.Split(baseURL.Hostname(), ".")
		for i := 0; i < len(parts)-1; i++ {
			for j := i + 1; j <= len(parts); j++ {
				substr := strings.Join(parts[i:j], ".")
				tempDirList = append(tempDirList, substr+".tar.gz", substr+".zip")
			}
		}
	} else {
		// is a ip
		tempDirList = append(tempDirList, baseURL.Hostname()+".tar.gz", baseURL.Hostname()+".zip")
	}
	for _, _path := range tempDirList {
		fullURL := baseURL.ResolveReference(&url.URL{Path: path.Join(baseURL.Path, _path)})
		finger, headers, respBody, statusCode := FingerScan(fullURL.String())
		if statusCode == 200 || statusCode == 500 {
			ret := CheckFinger(finger, fullURL.String(), headers, respBody, statusCode)
			if ret != "" {
				Info(fullURL.String() + " " + ret)
				result = append(result, ret)
			}
		}
	}
	return result
}
