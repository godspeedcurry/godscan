package utils

import (
	"fmt"
	"html"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"sync"

	"github.com/godspeedcurry/godscan/common"
	prettytable "github.com/jedib0t/go-pretty/v6/table"
	"github.com/jedib0t/go-pretty/v6/text"
	"github.com/mfonda/simhash"
	"github.com/spf13/viper"
)

const (
	INT_MAX = int(^uint(0) >> 1)
	INT_MIN = ^INT_MAX
)

func Max(nums ...int) int {
	var maxNum int = INT_MIN
	for _, num := range nums {
		if num > maxNum {
			maxNum = num
		}
	}
	return maxNum
}

func StringListToInterfaceList(tmpList []string) []interface{} {
	vals := make([]interface{}, len(tmpList))
	for i, v := range tmpList {
		vals[i] = v
	}
	return vals
}

func StatusColorTransformer(val interface{}) string {
	s := fmt.Sprintf("%v", val)
	code, err := strconv.Atoi(strings.TrimSpace(s))
	if err != nil {
		return s
	}
	switch {
	case code >= 500:
		return text.FgRed.Sprintf("%d", code)
	case code >= 400:
		return text.FgYellow.Sprintf("%d", code)
	case code >= 300:
		return text.FgHiYellow.Sprintf("%d", code)
	default:
		return text.FgHiGreen.Sprintf("%d", code)
	}
}

func Normalize(Path string, RootPath string) string {
	if strings.Contains(Path, "javascript:") {
		return ""
	} else if strings.HasPrefix(Path, "http://") {
		return Path
	} else if strings.HasPrefix(Path, "https://") {
		return Path
	} else if strings.HasPrefix(Path, "./") {
		return RootPath + Path[1:]
	} else if strings.HasPrefix(Path, "//") {
		return "https://" + strings.Replace(Path, "//", "", 1)
	} else if strings.HasPrefix(Path, "/") {
		return RootPath + Path
	} else {
		return RootPath + "/" + Path
	}
}

type sliceError struct {
	msg string
}

func (e *sliceError) Error() string {
	return e.msg
}

func Errorf(format string, args ...interface{}) error {
	msg := fmt.Sprintf(format, args...)
	return &sliceError{msg}
}

func RemoveDuplicateElement(originals interface{}) (interface{}, error) {
	temp := map[string]struct{}{}
	switch slice := originals.(type) {
	case []string:
		result := make([]string, 0, len(originals.([]string)))
		for _, item := range slice {
			key := fmt.Sprint(item)
			if _, ok := temp[key]; !ok {
				temp[key] = struct{}{}
				result = append(result, item)
			}
		}
		return result, nil
	case []int64:
		result := make([]int64, 0, len(originals.([]int64)))
		for _, item := range slice {
			key := fmt.Sprint(item)
			if _, ok := temp[key]; !ok {
				temp[key] = struct{}{}
				result = append(result, item)
			}
		}
		return result, nil
	case []int:
		result := make([]int, 0, len(originals.([]int)))
		for _, item := range slice {
			key := fmt.Sprint(item)
			if _, ok := temp[key]; !ok {
				temp[key] = struct{}{}
				result = append(result, item)
			}
		}
		return result, nil
	default:
		err := Errorf("Unknown type: %T", slice)
		return nil, err
	}
}

func ShowInfo() {
	Info("--suffix '%s'", strings.Join(common.SuffixTop, ","))
	Info("--prefix '%s'", strings.Join(common.PrefixTop, ","))
	Info("--sep '%s'", strings.Join(common.SeparatorTop, ","))
	Info("-k '%s'", strings.Join(common.KeywordTop, ","))
}
func RemoveDuplicatesString(arr []string) []string {
	// 创建一个空的map，用于存储唯一的元素
	uniqueMap := make(map[string]bool)
	result := []string{}

	// 遍历数组中的每个元素
	for _, ele := range arr {
		ele := strings.TrimSpace(ele)
		if strings.HasPrefix(ele, "====") {
			continue
		}
		// 将元素添加到map中，键为元素的值，值为true
		if !uniqueMap[ele] {
			uniqueMap[ele] = true
			result = append(result, ele)
		}
	}
	sort.Strings(result)
	return result
}

func Quote(x string) string {
	keys := []string{"+", "*", "[", "]", "(", ")", "?", ".", "{", "}"}
	for _, key := range keys {
		x = strings.ReplaceAll(x, key, "\\"+key)
	}
	return x
}

func SimHash(input []byte) uint64 {
	return simhash.Simhash(simhash.NewWordFeatureSet(input))
}

func UrlFormated(lines []string) []string {
	ret := []string{}
	for _, key := range lines {
		if strings.HasPrefix(key, "http") {
			ret = append(ret, key)
			continue
		}
		ret = append(ret, "http://"+key)
		ret = append(ret, "https://"+key)
	}
	return RemoveDuplicatesString(ret)
}

func FileReadLine(filename string) []string {
	data, err := os.ReadFile(filename)
	if err != nil {
		Debug("Skip reading file %s: %v", filename, err)
		return []string{}
	}
	lines := strings.Split(strings.Trim(string(data), "\n"), "\n")
	return RemoveDuplicatesString(lines)
}

func FilReadUrl(filename string) []string {
	lines := FileReadLine(filename)
	return UrlFormated(lines)
}

var fileMutex sync.Mutex

func FileWrite(filename string, format string, args ...interface{}) {
	if filename == "" {
		return
	}
	fileMutex.Lock()
	defer fileMutex.Unlock()

	dir := filepath.Dir(filename)
	err := os.MkdirAll(dir, os.ModePerm)
	if err != nil {
		fmt.Fprintf(os.Stderr, "file write mkdir failed: %v\n", err)
		return
	}

	file, err := os.OpenFile(filename, os.O_APPEND|os.O_WRONLY|os.O_CREATE, 0644)
	if err != nil {
		fmt.Fprintf(os.Stderr, "file open failed: %v\n", err)
		return
	}
	file.WriteString(fmt.Sprintf(format, args...))
}

func AddDataToTable(table prettytable.Writer, data []string) {
	if len(data) == 0 {
		return
	}
	row := make(prettytable.Row, len(data))
	for i := range data {
		row[i] = data[i]
	}
	table.AppendRow(row)
}

func extractText(htmlContent string) string {
	// 移除所有的脚本和样式内容
	re := regexp.MustCompile(`(?s)<(script|style)[^>]*>.*?</(script|style)>`)
	htmlContent = re.ReplaceAllString(htmlContent, "")

	// 移除所有的HTML标签
	re = regexp.MustCompile(`(?s)<[^>]*>`)
	text := re.ReplaceAllString(htmlContent, "")

	// 解码HTML实体
	text = html.UnescapeString(text)

	// 去除多余的空白字符
	text = strings.TrimSpace(text)
	text = regexp.MustCompile(`\s+`).ReplaceAllString(text, " ")

	return text
}

func getSelectedElements(content string) string {
	content = extractText(content)
	var re = regexp.MustCompile(`[0-9a-zA-Z_\-:\.\@]{4,15}`)
	arr := re.FindAllString(content, -1)
	n := len(arr)
	if n == 0 {
		return ""
	}
	var selected []string

	// 获取0元素
	selected = append(selected, arr[0])
	// 获取1/4元素
	selected = append(selected, arr[n>>2])
	// 获取中位数元素
	selected = append(selected, arr[n>>1])
	// 获取3/4元素
	selected = append(selected, arr[(n>>2)*3])
	// 获取last元素
	selected = append(selected, arr[n-1])

	// 确保元素唯一
	uniqueSelected := RemoveDuplicatesString(selected)
	return strings.Join(uniqueSelected, " ")
}

func CheckFinger(finger string, title string, Url string, contentType string, location string, respBody []byte, statusCode int) []string {
	if len(title) > 50 {
		title = title[:50] + "..."
	}
	hash := uint64(0)
	if location != "" {
		hash = SimHash([]byte(location))
	} else {
		hash = SimHash(respBody)
	}

	host, err := url.Parse(Url)
	if err != nil {
		Error("Error parse url %s", Url)
		return []string{}
	}
	host_port := host.Host
	if _, ok := fingerHashMap.Load(IpHash{host_port, hash}); !ok {
		fingerHashMap.Store(IpHash{host_port, hash}, true)
		return []string{Url, title, finger, contentType, strconv.Itoa(statusCode), location, strconv.Itoa(len(respBody)), getSelectedElements(string(respBody)), strconv.FormatUint(hash, 36)}
	}
	return []string{}
}

func WriteToCsv(filename string, data []string) {
	fileMutex.Lock()
	defer fileMutex.Unlock()

	// mirror to sqlite if available
	switch filename {
	case "finger.csv":
		SaveService("finger", data)
	case "dirbrute.csv":
		SaveService("dirbrute", data)
	}

	// Skip disk CSV artifacts to reduce intermediate files.
}

func SetHeaders(req *http.Request) {
	// 设置 User-Agent
	req.Header.Set("User-Agent", viper.GetString("DefaultUA"))

	// 设置自定义的请求头
	headers := viper.GetStringSlice("headers")
	for _, header := range headers {
		parts := strings.SplitN(header, ":", 2)
		if len(parts) != 2 {
			Warning("Invalid header format, correct format is 'Key: Value'")
			continue
		}
		key := strings.TrimSpace(parts[0])
		value := strings.TrimSpace(parts[1])
		req.Header.Set(key, value)
	}
}
