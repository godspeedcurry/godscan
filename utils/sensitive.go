package utils

import (
	"fmt"
	"html"
	"io"
	"math"
	"os"
	"regexp"
	"sort"
	"strings"

	"github.com/olekukonko/tablewriter"
)

type SensitiveData struct {
	Content string
	Entropy float64
}

func SecList() string {
	return "(?:" + strings.Join([]string{"secret", "secret[_-]?key", "token", "secret[_-]?token", "password",
		"aws[_-]?access[_-]?key[_-]?id", "aws[_-]?secret[_-]?access[_-]?key", "auth[-_]?token", "access[-_]?token",
		"auth[-_]?key", "client[-_]?secret", "access[-_]?key(?:secret|id)?",
		"id_dsa", "encryption[-_]?key", "passwd", "authorization", "bearer", "GITHUB[_-]?TOKEN",
		"api[_-]?key", "api[-_]?secret", "client[_-]?key", "client[_-]?id", "ssh[-_]?key",
		"ssh[-_]?key", "irc_pass", "xoxa-2", "xoxr", "private[_-]?key", "consumer[_-]?key", "consumer[_-]?secret",
		"SLACK_BOT_TOKEN", "api[-_]?token", "session[_-]?token", "session[_-]?key",
		"session[_-]?secret", "slack[_-]?token"}, "|") + ")"
}

// password = "123"
// password := "123"
// password === "123"
// password == "123"
// password != "123"
// "123" != password

func calculateEntropy(str string) float64 {
	charCount := make(map[rune]int)

	// 统计每个字符的出现次数
	for _, char := range str {
		charCount[char]++
	}

	entropy := 0.0
	strLength := len(str)

	// 计算每个字符在字符串中的概率，并计算熵
	for _, count := range charCount {
		probability := float64(count) / float64(strLength)
		entropy -= probability * math.Log2(probability)
	}

	return entropy
}

func DeduplicateByContent(data []SensitiveData) []SensitiveData {
	seen := make(map[string]bool)
	uniqueData := []SensitiveData{}

	for _, d := range data {
		if !seen[d.Content] {
			seen[d.Content] = true
			uniqueData = append(uniqueData, d)
		}
	}
	return uniqueData
}

func PrintTable(Url string, key string, data []SensitiveData) {
	Success("[%s] [%s]", Url, key)
	sort.Slice(data, func(i, j int) bool {
		return data[i].Entropy > data[j].Entropy
	})

	filename := "entropy.log"
	file, err := os.OpenFile(filename, os.O_APPEND|os.O_WRONLY|os.O_CREATE, 0644)
	if err != nil {
		fmt.Println("Failed to open file:", err)
		return
	}
	defer file.Close()

	FileWrite(filename, "[%s] [%s]\n", Url, key)

	multiWriter := io.MultiWriter(os.Stdout, file)
	table := tablewriter.NewWriter(multiWriter)

	table.SetHeader([]string{"Content", "Entropy"})

	for _, d := range data {
		row := []string{d.Content, fmt.Sprintf("%.2f", d.Entropy)}
		table.Append(row)
	}
	if table.NumLines() >= 1 {
		table.Render()
	}
}
func UrlFilter(Url string) bool {
	ignoreList := []string{
		"w3.org", ".woff", ".png", "apache.org", "gstatic.com", "google.com", "microsoft.com", ".gif", ".svg",
	}
	for _, ignore := range ignoreList {
		if strings.Contains(Url, ignore) {
			return true
		}
	}
	return false
}
func SensitiveInfoCollect(Url string, Content string) {
	space := `\s{0,5}`
	mustQuote := "['\"`]"
	quote := "['\"`]?"
	content := `([\w\.\!\@\#\$\%\^\&\*\~\-\+]{2,100})`
	// x := '123456', x = '123456', x == '123456' x === '123456'  x !== '123456' x != '123456'
	equals := `(:=|=|==|===|!==|!=|:)`
	// '123456' == x '123456' === x  '123456' !== x  '123456' != x
	equalss := `(==|===|!==|!=)`
	sec := SecList()
	infoMap := map[string]string{
		"Chinese Mobile Number": `[^\d]((?:(?:\+|00)86)?1(?:(?:3[\d])|(?:4[5-79])|(?:5[0-35-9])|(?:6[5-7])|(?:7[0-8])|(?:8[\d])|(?:9[189]))\d{8})[^\d]`,
		"Internal IP Address":   `[^0-9]((127\.0\.0\.1)|(10\.([0-1]?[0-9]{1,2}|2[0-4][0-9]|25[0-5])\.([0-1]?[0-9]{1,2}|2[0-4][0-9]|25[0-5])\.([0-1]?[0-9]{1,2}|2[0-4][0-9]|25[0-5]))|(172\.((1[6-9]|2[0-9]|3[0-1]))\.([0-1]?[0-9]{1,2}|2[0-4][0-9]|25[0-5])\.([0-1]?[0-9]{1,2}|2[0-4][0-9]|25[0-5]))|(192\.168\.([0-1]?[0-9]{1,2}|2[0-4][0-9]|25[0-5])\.([0-1]?[0-9]{1,2}|2[0-4][0-9]|25[0-5])))`,
		"Url":                   `((https?|ftp)://(?:[^\s:@/]+(?::[^\s:@/]*)?@)?[\w_\-\.]{5,100}(?::\d+)?(?:[/?][\w_\-\&\#/\.%]*)?)`,
		// 内容在右边
		"security-rule-0": `(?i)` + `(` + quote + sec + quote + space + equals + space + mustQuote + content + mustQuote + `)`,
		// 内容在左边
		"security-rule-1": `(?i)` + `(` + mustQuote + content + mustQuote + space + equalss + space + quote + sec + quote + `)`,
	}

	for key := range infoMap {
		reg := regexp.MustCompile(infoMap[key])
		res := reg.FindAllStringSubmatch(html.UnescapeString(Content), -1)
		secData := []SensitiveData{}
		otherData := []string{}
		if len(res) > 0 {
			for _, tmp := range res {
				if len(tmp) >= 4 {
					if key == "security-rule-0" {
						entropy := calculateEntropy(tmp[3])
						secData = append(secData, SensitiveData{Content: tmp[1], Entropy: entropy})
					} else {
						entropy := calculateEntropy(tmp[2])
						secData = append(secData, SensitiveData{Content: tmp[1], Entropy: entropy})
					}
				} else {
					if !UrlFilter(tmp[1]) {
						otherData = append(otherData, tmp[1])
					}
				}
			}
			otherData = removeDuplicatesString(otherData)
			if len(otherData) > 0 && len(otherData) < 20 {
				Success("[%s] [%s]\n%s", Url, key, strings.Join(otherData, "\n"))
			} else {
				Success("🌲🌲🌲 More info at ./result.log, found %d urls", len(otherData))
				FileWrite("result.log", "[%s] [%s]\n%s", Url, key, strings.Join(otherData, "\n"))
			}
			if len(secData) > 0 {
				PrintTable(Url, key, DeduplicateByContent(secData))
			}
		}
	}
}
