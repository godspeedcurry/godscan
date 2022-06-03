package utils

import (
	"log"
	"net/http"
	"net/url"

	"github.com/PuerkitoBio/goquery"
	"github.com/fatih/color"
)

func HttpGetServerHeader(Url string, NeedTitle bool) (string, string, error) {
	req, _ := http.NewRequest(http.MethodGet, Url, nil)
	req.Header.Set("User-Agent", "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/102.0.5005.61 Safari/537.36")
	resp, err := Client.Do(req)
	if err != nil {
		return "", "", err
	}
	defer resp.Body.Close()
	doc, err := goquery.NewDocumentFromReader(resp.Body)
	title := doc.Find("title").Text()
	if err != nil {
		log.Fatal(err)
	}
	ServerValue := resp.Header["Server"]
	if len(ServerValue) != 0 {
		return ServerValue[0], title, nil
	}
	return "", "", nil
}

func DisplayHeader(u string) {
	ServerHeader, Title, err := HttpGetServerHeader(u, true)
	if err != nil {
		color.HiRed("Error: %s\n", err)
	} else {
		color.Cyan("Server: %s\n", ServerHeader)
		color.Cyan("Title: %s\n", Title)
	}
}
func PrintFinger(Url string) {
	InitHttp()
	color.HiRed("Your URL: %s\n", Url)
	u, _ := url.Parse(Url)

	FirstUrl := u.Scheme + "://" + u.Hostname()
	DisplayHeader(FirstUrl)

	SecondUrl := u.Scheme + "://" + u.Hostname() + "/xxxxxx"
	DisplayHeader(SecondUrl)

}
