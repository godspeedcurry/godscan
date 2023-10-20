package utils

import (
	"fmt"
	"io/ioutil"
	"net"
	"net/http"
	"net/url"
	"strings"
	"sync"

	"github.com/fatih/color"
	"github.com/godspeedcurry/godscan/common"
	"github.com/gosuri/uiprogress"
)

var fingerHashMap = make(map[uint64]bool)
var result = []string{}

func formatUrl(raw string) string {
	if !strings.HasPrefix(raw, "http") {
		raw = "http://" + raw
	}
	return strings.TrimSpace(raw)
}

func DirBrute(filename string) {
	InitHttp()
	data, err := ioutil.ReadFile(filename)
	if err != nil {
		Error(err.Error())
		return
	}
	lines := strings.Split(strings.Trim(string(data), "\n"), "\n")
	lines = removeDuplicatesString(lines)

	uiprogress.Start()
	var wg sync.WaitGroup
	bar := uiprogress.AddBar(len(lines)).AppendCompleted().PrependElapsed()
	Info("Total: %d urls", len(lines))
	for _, line := range lines {
		wg.Add(1)
		go func(line string) {
			defer wg.Done()
			DirSingleBrute(line)
			bar.Incr()
		}(line)
	}
	wg.Wait()
	uiprogress.Stop()
	Success(color.GreenString("\n" + strings.Join(result, "\n")))
}

func CheckFinger(url string, statusCode int, length int, hash uint64) {
	ret := fingerScan(url)
	if !fingerHashMap[hash] {
		fingerHashMap[hash] = true
		result = append(result, fmt.Sprintf("[%d] %d {%s} %s", statusCode, length, ret, url))
	}
}

func DirSingleBrute(baseUrl string) {
	baseUrl = formatUrl(baseUrl)
	// 检查URL的存活性
	req, err := http.NewRequest(http.MethodGet, baseUrl, nil)
	if err != nil {
		Failed(baseUrl + " " + err.Error())
		return
	}
	req.Header.Set("User-Agent", common.DEFAULT_UA)
	resp, err := Client.Do(req)

	if err != nil {
		Failed(baseUrl + " " + err.Error())
		return
	}
	respBody, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		Failed(baseUrl + " " + err.Error())
		return
	}
	CheckFinger(baseUrl, resp.StatusCode, len(respBody), SimHash(respBody))

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
		req, err := http.NewRequest(http.MethodGet, fullURL.String(), nil)
		if err != nil {
			continue
		}
		req.Header.Set("User-Agent", common.DEFAULT_UA)
		resp, err := Client.Do(req)
		if err != nil {
			continue
		}
		respBody, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			continue
		}
		if resp.StatusCode == 200 || resp.StatusCode == 500 {
			CheckFinger(fullURL.String(), resp.StatusCode, len(respBody), SimHash(respBody))
		}
	}
	resp.Body.Close()
}
