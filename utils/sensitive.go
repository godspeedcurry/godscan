package utils

import (
	"fmt"
	"html"
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

func PrintTable(data []SensitiveData) {
	sort.Slice(data, func(i, j int) bool {
		return data[i].Entropy > data[j].Entropy
	})

	table := tablewriter.NewWriter(os.Stdout)
	table.SetHeader([]string{"Content", "Entropy"})

	for _, d := range data {
		row := []string{d.Content, fmt.Sprintf("%.2f", d.Entropy)}
		table.Append(row)
	}
	if table.NumLines() >= 1 {
		table.Render()
	}

}

func SensitiveInfoCollect(Url string, Content string) {
	space := `\s{0,5}`
	mustQuote := "['\"`]"
	quote := "['\"`]?"
	content := `([\w\+\_\-\/\=\$\!]{2,100})`
	equals := `[=!:]{1,3}`
	sec := SecList()
	infoMap := map[string]string{
		"Chinese Mobile Number": `[^\d]((?:(?:\+|00)86)?1(?:(?:3[\d])|(?:4[5-79])|(?:5[0-35-9])|(?:6[5-7])|(?:7[0-8])|(?:8[\d])|(?:9[189]))\d{8})[^\d]`,
		"Internal IP Address":   `[^0-9]((127\.0\.0\.1)|(10\.\d{1,3}\.\d{1,3}\.\d{1,3})|(172\.((1[6-9])|(2\d)|(3[01]))\.\d{1,3}\.\d{1,3})|(192\.168\.\d{1,3}\.\d{1,3}))`,
		"security-rule-0":       `(?i)` + `(` + quote + sec + quote + space + equals + space + mustQuote + content + mustQuote + `)`,
		"security-rule-1":       `(?i)` + `(` + mustQuote + content + mustQuote + space + equals + space + quote + sec + quote + `)`,
	}

	for key := range infoMap {
		reg := regexp.MustCompile(infoMap[key])
		res := reg.FindAllStringSubmatch(html.UnescapeString(Content), -1)
		secData := []SensitiveData{}
		otherDta := []string{}
		if len(res) > 0 {
			for _, tmp := range res {
				if len(tmp) >= 3 {
					entropy := calculateEntropy(tmp[2])
					secData = append(secData, SensitiveData{Content: tmp[1], Entropy: entropy})
				} else {
					otherDta = append(otherDta, tmp[1])
				}
			}
			otherDta = removeDuplicatesString(otherDta)
			if len(otherDta) > 0 {
				Success("[%s] [%s]\n%s", Url, key, strings.Join(otherDta, "\n"))
			}
			if len(secData) > 0 {
				Success("[%s] [%s]", Url, key)
				PrintTable(DeduplicateByContent(secData))
			}
		}
	}
}
