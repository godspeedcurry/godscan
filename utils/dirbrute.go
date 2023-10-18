package utils

import (
	"bufio"
	"fmt"
	"io/ioutil"
	"net"
	"net/http"
	"net/url"
	"os"
	"strings"

	"github.com/godspeedcurry/godscan/common"
)

func DirBrute(filename string) {
	urlFile, err := os.Open(filename)
	if err != nil {
		Warning(err.Error())
		return
	}
	defer urlFile.Close()

	scanner := bufio.NewScanner(urlFile)

	for scanner.Scan() {
		baseUrl := scanner.Text()
		DirSingleBrute(baseUrl)
	}
}

func DirSingleBrute(baseUrl string) {
	InitHttp()
	if !strings.HasPrefix(baseUrl, "http") {
		baseUrl = "http://" + baseUrl
	}
	statusCode := make(map[int]int)
	baseUrl = strings.TrimSpace(baseUrl)
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
	Success("[%d] %d {%s} %s", resp.StatusCode, len(respBody), fingerScan(baseUrl), baseUrl)
	statusCode[resp.StatusCode] += 1

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
		tempDirList = append(tempDirList, baseURL.Hostname()+".tar.gz")
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
		n := len(respBody)
		if resp.StatusCode == 200 || resp.StatusCode == 500 {
			Success("[%d] %d {%s} %s", resp.StatusCode, n, fingerScan(fullURL.String()), fullURL)
		}
		statusCode[resp.StatusCode] += 1
	}
	l := []string{}
	for key, value := range statusCode {
		l = append(l, fmt.Sprintf("%d, %d个", key, value))
	}
	Info("状态码: " + strings.Join(l, "|"))
	resp.Body.Close()
}
