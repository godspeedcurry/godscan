package utils

import (
	"database/sql"
	"fmt"
	"html"
	"math"
	"os"
	"regexp"
	"sort"
	"strings"

	prettytable "github.com/jedib0t/go-pretty/v6/table"
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
		"api[_-]?key", "api[-_]?secret", "client[_-]?key", "ssh[-_]?key",
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
	sort.Slice(data, func(i, j int) bool {
		return data[i].Entropy > data[j].Entropy
	})

	fmt.Printf("\n[%s] [%s]\n", Url, key)
	table := prettytable.NewWriter()
	table.SetOutputMirror(os.Stdout)
	table.SetStyle(prettytable.StyleRounded)
	table.SetColumnConfigs([]prettytable.ColumnConfig{
		{Number: 1, WidthMax: 80},
	})
	table.AppendHeader(prettytable.Row{"Content", "Entropy"})

	for _, d := range data {
		table.AppendRow(prettytable.Row{d.Content, fmt.Sprintf("%.2f", d.Entropy)})
	}
	if table.Length() >= 1 {
		table.Render()
	}
}
func UrlFilter(Url string) bool {
	ignoreList := []string{
		"w3.org", "apache.org", "gstatic.com", "google.com", "microsoft.com", "github.com", ".png", ".gif", ".woff",
	}
	for _, ignore := range ignoreList {
		if strings.Contains(Url, ignore) {
			return true
		}
	}
	return false
}
func SensitiveInfoCollect(db *sql.DB, Url string, Content string, directory string) {
	space := `[\s]{0,30}`
	mustQuote := "['\"`]"
	quote := "['\"`]?"
	content := `([\w\.\!\@\#\$\%\^\&\*\~\-\+ \=]{2,500})`
	// x := '123456', x = '123456', x == '123456' x === '123456'  x !== '123456' x != '123456'
	equals := `(:=|=|==|===|!==|!=|:)`
	// '123456' == x '123456' === x  '123456' !== x  '123456' != x
	equalss := `(==|===|!==|!=)`
	sec := SecList()

	infoMap := make(map[string]string)
	infoMap["Chinese Mobile Number"] = `[^\d]((?:(?:\+|00)86)?1(?:(?:3[\d])|(?:4[5-79])|(?:5[0-35-9])|(?:6[5-7])|(?:7[0-8])|(?:8[\d])|(?:9[189]))\d{8})[^\d]`
	infoMap["Internal IP Address"] = `[^0-9]((10\.([0-1]?[0-9]{1,2}|2[0-4][0-9]|25[0-5])\.([0-1]?[0-9]{1,2}|2[0-4][0-9]|25[0-5])\.([0-1]?[0-9]{1,2}|2[0-4][0-9]|25[0-5]))|(172\.((1[6-9]|2[0-9]|3[0-1]))\.([0-1]?[0-9]{1,2}|2[0-4][0-9]|25[0-5])\.([0-1]?[0-9]{1,2}|2[0-4][0-9]|25[0-5]))|(192\.168\.([0-1]?[0-9]{1,2}|2[0-4][0-9]|25[0-5])\.([0-1]?[0-9]{1,2}|2[0-4][0-9]|25[0-5])))`
	infoMap["Url"] = `((https?|ftp)://(?:[^\s:@/]+(?::[^\s:@/]*)?@)?[\w_\-\.]{5,256}(?::\d+)?(?:[/?][\w_\-\&\#/%.]*)?)`
	// 内容在右边
	infoMap["security-rule-0"] = `(?i)` + `(` + quote + sec + quote + space + equals + space + mustQuote + content + mustQuote + `)`
	// 内容在左边
	infoMap["security-rule-1"] = `(?i)` + `(` + mustQuote + content + mustQuote + space + equalss + space + quote + sec + quote + `)`

	for key := range infoMap {
		reg := regexp.MustCompile(infoMap[key])
		res := reg.FindAllStringSubmatch(html.UnescapeString(Content), -1)
		secData := []SensitiveData{}
		otherData := []string{}
		if len(res) > 0 {
			for _, tmp := range res {
				if key == "security-rule-0" {
					entropy := calculateEntropy(tmp[3])
					secData = append(secData, SensitiveData{Content: tmp[1], Entropy: entropy})
				} else if key == "security-rule-1" {
					entropy := calculateEntropy(tmp[2])
					secData = append(secData, SensitiveData{Content: tmp[1], Entropy: entropy})
				} else {
					otherData = append(otherData, tmp[0])
				}
			}
			otherData = RemoveDuplicatesString(otherData)
			if len(otherData) > 0 {
				FileWrite(directory+"urls.txt", "======[%s]======[%s]\n%s", Url, key, strings.Join(otherData, "\n")+"\n")
				InfoFile("[%s] [%s] %d item(s) saved to ./%s\n%s", Url, key, len(otherData), directory+"urls.txt", strings.Join(otherData, "\n"))
				SaveSensitiveHits(db, Url, key, otherData, directory)
			}
			if len(secData) > 0 {
				dedup := DeduplicateByContent(secData)
				PrintTable(Url, key, dedup)
				contents := []string{}
				entList := []EntropyHit{}
				for _, d := range dedup {
					contents = append(contents, d.Content)
					entList = append(entList, EntropyHit{
						SourceURL: Url,
						Category:  key,
						Content:   d.Content,
						Entropy:   d.Entropy,
						SaveDir:   directory,
					})
				}
				SaveSensitiveHits(db, Url, key, contents, directory)
				SaveEntropyHits(db, Url, key, directory, entList)
			}
		}
	}
}
