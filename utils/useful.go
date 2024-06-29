package utils

import (
	"encoding/csv"
	"fmt"
	"html"
	"math/rand"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/godspeedcurry/godscan/common"
	"github.com/mfonda/simhash"
	"github.com/olekukonko/tablewriter"
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

func Min(nums ...int) int {
	var minNum int = INT_MAX
	for _, num := range nums {
		if num < minNum {
			minNum = num
		}
	}
	return minNum
}

type Pair struct {
	Key   string
	Value int
}

type PairList []Pair

func (p PairList) Len() int           { return len(p) }
func (p PairList) Less(i, j int) bool { return p[i].Value < p[j].Value }
func (p PairList) Swap(i, j int)      { p[i], p[j] = p[j], p[i] }

func StringListToInterfaceList(tmpList []string) []interface{} {
	vals := make([]interface{}, len(tmpList))
	for i, v := range tmpList {
		vals[i] = v
	}
	return vals
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

func RandomString(length int) string {
	// 定义字符集
	charset := "0123456789abcdef"

	// 初始化随机数生成器
	rand.NewSource(time.Now().UnixNano())

	// 生成随机字符
	result := make([]byte, length)
	for i := range result {
		result[i] = charset[rand.Intn(len(charset))]
	}

	return string(result)
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
	fmt.Println("--suffix '" + strings.Join(common.SuffixTop, ",") + "'")
	fmt.Println("--prefix '" + strings.Join(common.PrefixTop, ",") + "'")
	fmt.Println("--sep '" + strings.Join(common.SeparatorTop, ",") + "'")
	fmt.Println("-k '" + strings.Join(common.KeywordTop, ",") + "'")
}
func RemoveDuplicatesString(arr []string) []string {
	// 创建一个空的map，用于存储唯一的元素
	uniqueMap := make(map[string]bool)
	result := []string{}

	// 遍历数组中的每个元素
	for _, ele := range arr {
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
		Error("Error read file %s", filename)
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
	fileMutex.Lock()
	defer fileMutex.Unlock()

	dir := filepath.Dir(filename)
	err := os.MkdirAll(dir, os.ModePerm)
	if err != nil {
		Fatal("%s", err)
		return
	}

	file, err := os.OpenFile(filename, os.O_APPEND|os.O_WRONLY|os.O_CREATE, 0644)
	if err != nil {
		Fatal("%s", err)
		return
	}
	file.WriteString(fmt.Sprintf(format, args...))
}

func AddDataToTable(table *tablewriter.Table, data []string) {
	if len(data) > 0 {
		table.Append(data)
	}
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

func CheckFinger(finger string, title string, Url string, contentType string, respBody []byte, statusCode int) []string {
	if len(title) > 50 {
		title = title[:50] + "..."
	}
	hash := SimHash(respBody)

	host, err := url.Parse(Url)
	if err != nil {
		Error("Error parse url %s", Url)
	}
	host_port := host.Host
	if _, ok := fingerHashMap[IpHash{host_port, hash}]; !ok {
		fingerHashMap[IpHash{host_port, hash}] = true
		return []string{Url, title, finger, contentType, strconv.Itoa(statusCode), strconv.Itoa(len(respBody)), strconv.FormatUint(hash, 16), getSelectedElements(string(respBody))}
	}
	return []string{}
}

func WriteToCsv(filename string, data []string) {
	file, err := os.OpenFile(filename, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		panic(err)
	}
	defer file.Close()

	header := []string{"Url", "Title", "Finger", "Content-Type", "StatusCode", "Length", "SimHash", "Keyword"}
	fileInfo, err := file.Stat()
	writer := csv.NewWriter(file)

	if err != nil {
		panic(err)
	}
	if fileInfo.Size() == 0 {
		if err := writer.Write(header); err != nil {
			panic(err) // 处理写入错误
		}
	}

	err = writer.Write(data)
	if err != nil {
		panic(err)
	}
	writer.Flush()
}
