package utils

import (
	"bufio"
	"fmt"
	"io/ioutil"
	"log"
	"main/common"
	"net"
	"net/http"
	"net/url"
	"os"
	"strings"

	"github.com/fatih/color"
)

func DirBrute(filename string) {
	urlFile, err := os.Open(filename)
	if err != nil {
		log.Fatal(err)
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
	baseUrl = strings.TrimSpace(baseUrl)
	// 检查URL的存活性
	req, _ := http.NewRequest(http.MethodGet, baseUrl, nil)
	req.Header.Set("User-Agent", common.DEFAULT_UA)
	resp, err := Client.Do(req)

	if err != nil {
		color.Red("[failed] %s: %v\n", baseUrl, err)
		return
	}
	color.White("[*] %s is alive, start to brute...\n", baseUrl)

	baseURL, _ := url.Parse(baseUrl)
	tempDirList := common.DirList

	ip := net.ParseIP(baseURL.Hostname())
	if ip == nil {
		// is a domain
		parts := strings.Split(baseURL.Hostname(), ".")

		for i := 0; i < len(parts)-1; i++ {
			for j := i + 1; j <= len(parts); j++ {
				substr := strings.Join(parts[i:j], ".")
				tempDirList = append(tempDirList, substr+".tar.gz")
			}
		}
	} else {
		// is a ip
		tempDirList = append(tempDirList, baseURL.Hostname()+".tar.gz")
	}

	for _, _path := range tempDirList {
		fullURL := baseURL.ResolveReference(&url.URL{Path: _path})
		req, _ := http.NewRequest(http.MethodGet, fullURL.String(), nil)
		req.Header.Set("User-Agent", common.DEFAULT_UA)
		resp, err := Client.Do(req)
		if err != nil {
			fmt.Println(err)
			return
		}
		respBody, err := ioutil.ReadAll(resp.Body)

		if resp.StatusCode == 200 && len(respBody) > 0 {
			color.HiGreen("[success] %s len=%d status_code=%d finger=%s", fullURL, len(respBody), resp.StatusCode, fingerScan(fullURL.String()))
		} else if resp.StatusCode == 500 {
			color.HiGreen("[warning] %s len=%d status_code=%d", fullURL, len(respBody), resp.StatusCode)
		} else {
			color.HiBlue("[info] %s len=%d status_code=%d", fullURL, len(respBody), resp.StatusCode)
		}
	}
	resp.Body.Close()
}
