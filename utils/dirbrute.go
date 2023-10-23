package utils

import (
	"fmt"
	"net"
	"net/http"
	"net/url"
	"strings"

	"github.com/godspeedcurry/godscan/common"
	"github.com/spf13/viper"
)

var fingerHashMap = make(map[uint64]bool)
var result = []string{}

func formatUrl(raw string) string {
	if !strings.HasPrefix(raw, "http") {
		raw = "http://" + raw
	}
	return strings.TrimSpace(raw)
}

func CheckFinger(finger string, url string, statusCode int, length int, hash uint64) string {
	if !fingerHashMap[hash] {
		fingerHashMap[hash] = true
		return fmt.Sprintf("[%d] %d {%s} %s", statusCode, length, finger, url)
	}
	return ""
}

func CheckAlive(url string) bool {
	// 检查URL的存活性
	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		Fatal(url + " " + err.Error())
	}
	req.Header.Set("User-Agent", viper.GetString("DefaultUA"))

	_, err = Client.Do(req)
	if err != nil {
		Fatal(url + " " + err.Error())
		return false
	}
	return true
}

func DirBrute(baseUrl string) []string {

	result := []string{}
	baseUrl = formatUrl(baseUrl)

	if CheckAlive(baseUrl) == false {
		return nil
	}

	baseURL, _ := url.Parse(baseUrl)
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
		fullURL := baseURL.ResolveReference(&url.URL{Path: _path})
		finger, respBody, statusCode := FingerScan(fullURL.String())

		if statusCode == 200 || statusCode == 500 {
			ret := CheckFinger(finger, fullURL.String(), statusCode, len(respBody), SimHash(respBody))
			if ret != "" {
				result = append(result, ret)
			}
		}
	}
	return result
}
