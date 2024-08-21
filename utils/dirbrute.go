package utils

import (
	"net/http"
	"net/url"
	"path"
	"strings"
	"sync"

	"github.com/fatih/color"
	"github.com/godspeedcurry/godscan/common"
	"github.com/spf13/viper"
)

type IpHash struct {
	Ip   string
	Hash uint64
}

// var fingerHashMap = make(map[IpHash]bool)
var fingerHashMap sync.Map

func formatUrl(raw string) string {
	if !strings.HasPrefix(raw, "http") {
		raw = "http://" + raw
	}
	return strings.TrimSpace(raw)
}

func DirBrute(baseUrl string, dir string) []string {
	result := []string{}
	baseURL, err := url.Parse(formatUrl(baseUrl))
	if err != nil {
		Error("%s", err)
		return []string{}
	}
	fullURL := baseURL.ResolveReference(&url.URL{Path: path.Join(baseURL.Path, dir)})
	if strings.HasSuffix(dir, "/") && dir != "/" {
		fullURL.Path += "/"
	}
	finger, _, title, contentType, location, respBody, statusCode := FingerScan(fullURL.String(), http.MethodGet, viper.GetBool("redirect"))
	if statusCode == 200 || statusCode == 500 || statusCode == 302 {
		result = CheckFinger(finger, title, fullURL.String(), contentType, location, respBody, statusCode)
	}
	if len(result) > 0 {
		WriteToCsv("dirbrute.csv", result)
	}

	if len(result) > 0 {
		if result[2] != common.NoFinger {
			result[2] = color.GreenString(result[2])
		}
		if strings.Contains(result[5], "nacos") {
			result[5] = color.GreenString(result[5])
		}
	}
	return result
}
