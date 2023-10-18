package utils

import (
	"fmt"
	"sort"
	"strings"

	"github.com/godspeedcurry/godscan/common"
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

func mysort(mymap map[string]int) PairList {
	pl := make(PairList, len(mymap))
	i := 0
	for k, v := range mymap {
		pl[i] = Pair{k, v}
		i++
	}
	sort.Sort(sort.Reverse(pl))
	return pl
}

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
	} else if strings.HasPrefix(Path, "/") {
		return RootPath + Path
	} else if strings.HasPrefix(Path, "//") {
		return strings.Replace(Path, "//", "", 1)
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
	default:
		err := Errorf("Unknown type: %T", slice)
		return nil, err
	}
}

func ShowInfo() {
	fmt.Println("-suffix '" + strings.Join(common.SuffixTop, ",") + "'")
	fmt.Println("-prefix '" + strings.Join(common.PrefixTop, ",") + "'")
	fmt.Println("-sep '" + strings.Join(common.SeparatorTop, ",") + "'")
	fmt.Println("-k '" + strings.Join(common.KeywordTop, ",") + "'")
}
func removeDuplicatesString(arr []string) []string {
	// 创建一个空的map，用于存储唯一的元素
	uniqueMap := make(map[string]bool)
	result := []string{}

	// 遍历数组中的每个元素
	for _, num := range arr {
		// 将元素添加到map中，键为元素的值，值为true
		if !uniqueMap[num] {
			uniqueMap[num] = true
			result = append(result, num)
		}
	}
	sort.Strings(result)
	return result
}

func in(str_array []string, target string) bool {
	sort.Strings(str_array)
	index := sort.SearchStrings(str_array, target)
	//index的取值：[0,len(str_array)]
	if index < len(str_array) && str_array[index] == target { //需要注意此处的判断，先判断 &&左侧的条件，如果不满足则结束此处判断，不会再进行右侧的判断
		return true
	}
	return false
}

func Quote(x string) string {
	keys := []string{"+", "*", "[", "]", "(", ")", "?", ".", "{", "}"}
	for _, key := range keys {
		x = strings.ReplaceAll(x, key, "\\"+key)
	}
	return x
}
