package utils

import (
	"fmt"
	// "github.com/chain-zhang/pinyin"
	"main/common"
	"regexp"
	"strings"

	"github.com/Chain-Zhang/pinyin"
)

func MightBeIdentityCard(IdentityCard string) bool {
	result, _ := regexp.MatchString(`^(\d{17})([0-9]|X|x)$`, IdentityCard)
	return result
}

func MightBeChineseName(Name string) bool {
	result, _ := regexp.MatchString("^[\u4e00-\u9fa5]*$", Name)
	// fmt.Println(result)
	return result
}

func MightBePhone(Phone string) bool {
	// 也有可能是qq号 生日什么的
	result, _ := regexp.MatchString(`^\d{6,11}$`, Phone)
	return result
}

func TranslateToEnglish(Name string) (string, string, string) {
	str, err := pinyin.New(Name).Split(" ").Mode(pinyin.WithoutTone).Convert()
	if err != nil {
		fmt.Println(err)
		return "", "", ""
	}
	// 首字母
	onlyFirst := ""
	// 姓全称其他只取首字母
	firstComplete := ""
	for idx, x := range strings.Split(str, " ") {
		onlyFirst = onlyFirst + string(x[0])
		if idx == 0 {
			firstComplete = firstComplete + x
		} else {
			firstComplete = firstComplete + string(x[0])
		}
	}
	return onlyFirst, firstComplete, strings.ReplaceAll(str, " ", "")
}

func FirstCharToUpper(Name string) string {
	return strings.ToUpper(Name[:1]) + Name[1:]
}

func LastCharToUpper(Name string) string {
	if len(Name) > 0 {
		return Name[:len(Name)-1] + strings.ToUpper(Name[len(Name)-1:])
	}
	return ""
}

func AddStringToString(x string, y string, seps []string) []string {
	var mylist = []string{}
	for _, first := range []string{x, FirstCharToUpper(x), strings.ToUpper(x), LastCharToUpper(x)} {
		for _, sep := range seps {
			for _, last := range []string{y, FirstCharToUpper(y), strings.ToUpper(y), LastCharToUpper(y)} {
				mylist = append(mylist, first+sep+last)
			}
		}
	}

	notDuplcated, _ := RemoveDuplicateElement(mylist)
	return notDuplcated.([]string)
}

func BuildFromKeyWordList(KeywordList []string) []string {
	var onlyFirst = "zs"
	var firstComplete = "zhangs"
	var Phone = "123456"
	var IdentityCard = "123456789123456789"
	var completeName = "zhangsan"
	for _, Keyword := range KeywordList {
		if MightBeChineseName((Keyword)) {
			onlyFirst, firstComplete, completeName = TranslateToEnglish(Keyword)
		} else if MightBeIdentityCard(Keyword) {
			IdentityCard = Keyword
		} else if MightBePhone(Keyword) {
			Phone = Keyword
		}
	}
	var ans = []string{}
	if Phone != "123456" {
		ans = append(ans, AddStringToString(Phone, onlyFirst, []string{"@", "_", "#", ""})...)
		ans = append(ans, AddStringToString(Phone, firstComplete, []string{"@", "_", "#", ""})...)
		ans = append(ans, AddStringToString(Phone, completeName, []string{"@", "_", "#", ""})...)
		ans = append(ans, AddStringToString(Phone[2:], onlyFirst, []string{"@", "_", "#", ""})...)
		ans = append(ans, AddStringToString(Phone[2:], firstComplete, []string{"@", "_", "#", ""})...)
		ans = append(ans, AddStringToString(Phone[2:], completeName, []string{"@", "_", "#", ""})...)
	}

	if IdentityCard != "123456789123456789" {
		ans = append(ans, AddStringToString(onlyFirst, IdentityCard[8:14], []string{"@", "_", "#", ""})...)
		ans = append(ans, AddStringToString(onlyFirst, IdentityCard[10:14], []string{"@", "_", "#", ""})...)
		ans = append(ans, AddStringToString(onlyFirst, IdentityCard[6:10], []string{"@", "_", "#", ""})...)
		ans = append(ans, AddStringToString(onlyFirst, IdentityCard[12:18], []string{"@", "_", "#", ""})...)
	}
	return ans
}

func GenerateWeakPassword(KeywordListStr string, SuffixListStr string) []string {
	var KeywordList = []string{}
	if strings.Contains(KeywordListStr, ",") {
		KeywordList = strings.Split(KeywordListStr, ",")
	} else {
		KeywordList = append(KeywordList, KeywordListStr)
	}
	var PasswordList = []string{}

	var SuffixList = []string{}
	if !(SuffixListStr == "") {
		SuffixList = strings.Split(SuffixListStr, ",")
	}

	for _, keyword := range KeywordList {
		for _, sep := range SuffixList {
			PasswordList = append(PasswordList, AddStringToString(keyword, sep, []string{"@", "_", "#", "!@#", "", "123", "qwe"})...)
		}
	}

	for _, password := range common.Passwords {
		if !strings.Contains(password, "{user}") {
			PasswordList = append(PasswordList, password)
			continue
		}
		for _, keyword := range KeywordList {

			//如果是身份证格式
			if MightBeIdentityCard(keyword) {
				// 2009
				PasswordList = append(PasswordList, strings.ReplaceAll(password, "{user}", keyword[6:10]))
				// 后六位
				PasswordList = append(PasswordList, strings.ReplaceAll(password, "{user}", keyword[12:18]))
				// 年份后两位 + 生日
				PasswordList = append(PasswordList, strings.ReplaceAll(password, "{user}", keyword[8:14]))
				// 生日
				PasswordList = append(PasswordList, strings.ReplaceAll(password, "{user}", keyword[10:14]))
			} else if MightBeChineseName(keyword) {
				onlyFirst, firstComplete, completeName := TranslateToEnglish(keyword)
				PasswordList = append(PasswordList, strings.ReplaceAll(password, "{user}", onlyFirst))
				PasswordList = append(PasswordList, strings.ReplaceAll(password, "{user}", FirstCharToUpper(onlyFirst)))
				PasswordList = append(PasswordList, strings.ReplaceAll(password, "{user}", LastCharToUpper(onlyFirst)))
				PasswordList = append(PasswordList, strings.ReplaceAll(password, "{user}", strings.ToUpper(onlyFirst)))

				PasswordList = append(PasswordList, strings.ReplaceAll(password, "{user}", firstComplete))
				PasswordList = append(PasswordList, strings.ReplaceAll(password, "{user}", FirstCharToUpper(firstComplete)))
				PasswordList = append(PasswordList, strings.ReplaceAll(password, "{user}", LastCharToUpper(firstComplete)))
				PasswordList = append(PasswordList, strings.ReplaceAll(password, "{user}", strings.ToUpper(firstComplete)))

				PasswordList = append(PasswordList, strings.ReplaceAll(password, "{user}", completeName))
				PasswordList = append(PasswordList, strings.ReplaceAll(password, "{user}", FirstCharToUpper(completeName)))
				PasswordList = append(PasswordList, strings.ReplaceAll(password, "{user}", LastCharToUpper(completeName)))
				PasswordList = append(PasswordList, strings.ReplaceAll(password, "{user}", strings.ToUpper(completeName)))

			} else {
				PasswordList = append(PasswordList, strings.ReplaceAll(password, "{user}", keyword))
				PasswordList = append(PasswordList, strings.ReplaceAll(password, "{user}", FirstCharToUpper(keyword)))
				PasswordList = append(PasswordList, strings.ReplaceAll(password, "{user}", LastCharToUpper(keyword)))
				PasswordList = append(PasswordList, strings.ReplaceAll(password, "{user}", strings.ToUpper(keyword)))
			}
		}
	}

	PasswordList = append(PasswordList, BuildFromKeyWordList(KeywordList)...)
	UniqPasswordList, err := RemoveDuplicateElement(PasswordList)
	if err != nil {
		fmt.Println(err)
		return []string{}
	}
	fmt.Println(strings.Join(UniqPasswordList.([]string), "\n"))
	return UniqPasswordList.([]string)
}
