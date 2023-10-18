package utils

import (
	"encoding/json"
	"fmt"
	"log"
	"strconv"
	"time"

	"regexp"
	"strings"

	"github.com/godspeedcurry/godscan/common"

	"github.com/Chain-Zhang/pinyin"
	"github.com/Lofanmi/chinese-calendar-golang/calendar"
)

func MightBeIdentityCard(IdentityCard string) bool {
	result, _ := regexp.MatchString(`^(\d{17})([0-9]|X|x)$`, IdentityCard)
	return result
}

func MightBeChineseName(Name string) bool {
	result, _ := regexp.MatchString("^[\u4e00-\u9fa5]+$", Name)
	return result
}

func MightBePhone(Phone string) bool {
	// 也有可能是qq号 生日什么的
	result, _ := regexp.MatchString(`^\d{6,11}$`, Phone)
	return result
}

func TranslateToEnglish(Name string) (string, string, string, string) {
	str, err := pinyin.New(Name).Split(" ").Mode(pinyin.WithoutTone).Convert()
	if err != nil {
		fmt.Println(err)
		return "", "", "", ""
	}
	// 首字母
	onlyFirst := ""
	// 姓全称其他只取首字母
	firstComplete := ""
	// 所有拼音首字母大写
	firstUpper := ""
	for idx, x := range strings.Split(str, " ") {
		onlyFirst = onlyFirst + string(x[0])
		firstUpper = firstUpper + strings.Title(x)
		if idx == 0 {
			firstComplete = firstComplete + x
		} else {
			firstComplete = firstComplete + string(x[0])
		}
	}
	return onlyFirst, firstComplete, strings.ReplaceAll(str, " ", ""), firstUpper
}

func FirstCharToUpper(Name string) string {
	return strings.ToUpper(Name[:1]) + Name[1:]
}

func HalfCharToUpper(Name string) string {
	return strings.ToUpper(Name[:len(Name)>>1]) + Name[len(Name)>>1:]
}

func LastCharToUpper(Name string) string {
	if len(Name) > 0 {
		return Name[:len(Name)-1] + strings.ToUpper(Name[len(Name)-1:])
	}
	return ""
}

func AddStringToString(x string, seps []string, y string) []string {
	var mylist = []string{}
	for _, first := range []string{x, FirstCharToUpper(x), strings.ToUpper(x), LastCharToUpper(x)} {
		for _, sep := range seps {
			for _, last := range []string{y, FirstCharToUpper(y), strings.ToUpper(y), LastCharToUpper(y)} {
				mylist = append(mylist, first+sep+last)
			}
		}
	}
	notDuplicated, _ := RemoveDuplicateElement(mylist)
	return notDuplicated.([]string)
}

func getLunar(keyword string) string {
	year, _ := strconv.ParseInt(keyword[6:10], 10, 64)
	month, _ := strconv.ParseInt(keyword[10:12], 10, 64)
	day, _ := strconv.ParseInt(keyword[12:14], 10, 64)

	c := calendar.BySolar(year, month, day, 12, 0, 0)
	bytes, _ := c.ToJSON()

	var data map[string]interface{}
	err := json.Unmarshal(bytes, &data)
	if err != nil {
		log.Fatal(err)
	}
	// 获得农历生日
	lunar := fmt.Sprintf("%02d%02d", int(data["lunar"].(map[string]interface{})["month"].(float64)), int(data["lunar"].(map[string]interface{})["day"].(float64)))
	return lunar

}

func ReplaceWithTable(input string) string {
	var tmp = input
	for _, char := range tmp {
		switch char {
		case 'a':
			tmp = strings.ReplaceAll(tmp, string(char), "@")
		case 's':
			tmp = strings.ReplaceAll(tmp, string(char), "5")
		case 'o':
			tmp = strings.ReplaceAll(tmp, string(char), "0")
		case 'i':
			tmp = strings.ReplaceAll(tmp, string(char), "1")
		}
	}
	return tmp
}
func generateVariants(input string) []string {
	variants := []string{}
	variants = append(variants, strings.ReplaceAll(input, "a", "@"))
	// variants = append(variants, strings.ReplaceAll(input, "a", "4"))
	variants = append(variants, strings.ReplaceAll(input, "s", "5"))
	variants = append(variants, strings.ReplaceAll(input, "o", "0"))
	variants = append(variants, strings.ReplaceAll(input, "i", "1"))
	variants = append(variants, ReplaceWithTable(input))
	return variants
}

func getSuffixList(Info common.HostInfo) []string {
	var SuffixList = []string{""}
	if Info.Full {
		SuffixList = append(SuffixList, common.SuffixTop...)
	}
	year := time.Now().Year()
	for i := year - 15; i <= year; i++ {
		SuffixList = append(SuffixList, strconv.Itoa(i))
	}
	if Info.Full {
		for i := year - 50; i < year-15; i++ {
			SuffixList = append(SuffixList, strconv.Itoa(i))
		}
	}
	SuffixList = append(SuffixList, strings.Split(Info.Suffix, ",")...)
	SuffixList = removeDuplicatesString(SuffixList)
	return SuffixList
}

func getKeywordList(Info common.HostInfo) []string {
	var KeywordList = []string{"admin"}
	if Info.Full {
		KeywordList = append(KeywordList, common.KeywordTop...)
	}
	KeywordList = append(KeywordList, strings.Split(Info.Keywords, ",")...)
	KeywordList = removeDuplicatesString(KeywordList)
	return KeywordList
}

func getSepList(Info common.HostInfo) []string {
	var SepList = []string{""}
	if Info.Full {
		SepList = append(SepList, common.SeparatorTop...)
	}
	SepList = append(SepList, strings.Split(Info.Separator, ",")...)
	SepList = removeDuplicatesString(SepList)
	return SepList
}

func getPrefixList(Info common.HostInfo) []string {
	var PrefixList = []string{""}
	if Info.Full {
		PrefixList = append(PrefixList, common.PrefixTop...)
	}
	PrefixList = append(PrefixList, strings.Split(Info.Prefix, ",")...)
	PrefixList = removeDuplicatesString(PrefixList)
	return PrefixList
}

func processIdentityCard(keyword string) []string {
	arr := []string{}
	lunar := getLunar(keyword)
	arr = append(arr, keyword[6:10])

	// 生日
	arr = append(arr, keyword[10:14])

	// 后六位
	arr = append(arr, keyword[12:18])

	// 年份后两位 + 生日
	arr = append(arr, keyword[8:14])
	// 年份后两位 + 农历生日
	arr = append(arr, keyword[8:10]+lunar)

	// 年份 + 农历生日
	arr = append(arr, keyword[6:10]+lunar)
	return arr
}
func outputListFormat(UniqPasswordList []string, HostInfo common.HostInfo) {
	if common.ListFormat {
		var quotedStrings []string
		for _, str := range UniqPasswordList {
			quotedStrings = append(quotedStrings, strconv.Quote(str))
		}
		output := "[" + strings.Join(quotedStrings, ", ") + "]"
		fmt.Println(output)
	} else {
		fmt.Println(strings.Join(UniqPasswordList, "\n"))
	}

}
func GenerateWeakPassword(HostInfo common.HostInfo) []string {
	var PasswordList = []string{}
	var KeywordList = getKeywordList(HostInfo)
	var SuffixList = getSuffixList(HostInfo)
	var SepList = getSepList(HostInfo)
	var PrefixList = getPrefixList(HostInfo)

	var idcard, onlyFirst, firstComplete, completeName, firstUpper string

	PasswordList = append(PasswordList, common.Passwords...)

	KeywordTmpList := []string{}
	for _, keyword := range KeywordList {
		if MightBeIdentityCard(keyword) {
			idcard = keyword
			KeywordTmpList = append(KeywordList, processIdentityCard(keyword)...)

		} else if MightBeChineseName(keyword) {
			onlyFirst, firstComplete, completeName, firstUpper = TranslateToEnglish(keyword)
			// 也可以作为前后缀
			// SuffixList = append(SuffixList, completeName)
			names := []string{onlyFirst, firstComplete, completeName}
			for _, name := range names {
				KeywordTmpList = append(KeywordTmpList, name, FirstCharToUpper(name), LastCharToUpper(name), strings.ToUpper(name), HalfCharToUpper(name))
			}
			KeywordTmpList = append(KeywordTmpList, firstUpper)
			if HostInfo.Variant {
				KeywordTmpList = append(KeywordTmpList, generateVariants(completeName)...)
			}
		} else {
			KeywordTmpList = append(KeywordTmpList, keyword, FirstCharToUpper(keyword), LastCharToUpper(keyword), strings.ToUpper(keyword))
			if HostInfo.Variant {
				KeywordTmpList = append(KeywordTmpList, generateVariants(keyword)...)
			}
		}
	}
	fmt.Println(KeywordTmpList)
	fmt.Println("=====")
	for _, pre := range PrefixList {
		for _, keyword := range KeywordTmpList {
			for _, sep := range SepList {
				for _, suffix := range SuffixList {
					PasswordList = append(PasswordList, pre+keyword+sep+suffix)
				}
			}
		}
	}

	if idcard != "" && completeName != "" {
		arr := []string{onlyFirst, firstComplete, completeName, FirstCharToUpper(onlyFirst), LastCharToUpper(onlyFirst), strings.ToUpper(onlyFirst), FirstCharToUpper(firstComplete), LastCharToUpper(completeName), strings.ToUpper(completeName)}
		for _, k := range arr {
			PasswordList = append(PasswordList, k+idcard[10:14], k+idcard[8:14], k+idcard[12:18], k+idcard[6:10], k+idcard[6:14])
		}
	}
	UniqPasswordList, err := RemoveDuplicateElement(PasswordList)
	if err != nil {
		Error("%s", err)
		return []string{}
	}
	outputListFormat(UniqPasswordList.([]string), HostInfo)
	println("total:", len(UniqPasswordList.([]string)))
	return UniqPasswordList.([]string)
}
