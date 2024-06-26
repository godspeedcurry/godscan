package utils

import (
	"net/http"
	"net/url"
	"path"
	"strings"

	"github.com/fatih/color"
	"github.com/godspeedcurry/godscan/common"
)

type IpHash struct {
	Ip   string
	Hash uint64
}

var fingerHashMap = make(map[IpHash]bool)

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
	finger, _, title, contentType, respBody, statusCode := FingerScan(fullURL.String(), http.MethodGet)
	if statusCode == 200 || statusCode == 500 {
		result = CheckFinger(finger, title, fullURL.String(), contentType, respBody, statusCode)
	}
	if len(result) > 0 {
		WriteToCsv("dirbrute.csv", result)
	}

	if len(result) > 0 {
		if result[2] != common.NoFinger {
			result[2] = color.GreenString(result[2])
		}
	}
	return result
}
