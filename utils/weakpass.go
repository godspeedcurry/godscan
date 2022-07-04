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
	result, _ := regexp.MatchString(`^\d{4,11}$`, Phone)
	return result
}

func TranslateToEnglish(Name string) (string, string) {
	str, err := pinyin.New(Name).Split(" ").Mode(pinyin.WithoutTone).Convert()
	if err != nil {
		fmt.Println(err)
		return "", ""
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
	return onlyFirst, firstComplete
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
	for _, sep := range seps {
		mylist = append(mylist, x+sep+y)
		mylist = append(mylist, x+sep+y)
		mylist = append(mylist, FirstCharToUpper(x)+sep+FirstCharToUpper(y))
		mylist = append(mylist, FirstCharToUpper(x)+sep+y)
		mylist = append(mylist, x+sep+FirstCharToUpper(y))
		mylist = append(mylist, LastCharToUpper(x)+sep+LastCharToUpper(y))
		mylist = append(mylist, LastCharToUpper(x)+sep+y)
		mylist = append(mylist, x+sep+LastCharToUpper(y))
		mylist = append(mylist, y+sep+x)
		mylist = append(mylist, FirstCharToUpper(y)+sep+FirstCharToUpper(x))
		mylist = append(mylist, FirstCharToUpper(y)+sep+x)
		mylist = append(mylist, y+sep+FirstCharToUpper(x))
		mylist = append(mylist, LastCharToUpper(y)+sep+LastCharToUpper(x))
		mylist = append(mylist, LastCharToUpper(y)+sep+x)
		mylist = append(mylist, y+sep+LastCharToUpper(x))
	}

	notDuplcated, _ := RemoveDuplicateElement(mylist)
	return notDuplcated.([]string)
}

func BuildFromKeyWordList(KeywordList []string) []string {
	var onlyFirst = "zs"
	var firstComplete = "zhangs"
	var Phone = "123456"
	var IdentityCard = "123456789123456789"
	for _, Keyword := range KeywordList {
		if MightBeChineseName((Keyword)) {
			onlyFirst, firstComplete = TranslateToEnglish(Keyword)
		} else if MightBeIdentityCard(Keyword) {
			IdentityCard = Keyword
		} else if MightBePhone(Keyword) {
			Phone = Keyword
		}
	}
	var ans = []string{}
	ans = append(ans, AddStringToString(Phone, onlyFirst, []string{"@", "_", "#", ""})...)
	ans = append(ans, AddStringToString(Phone, firstComplete, []string{"@", "_", "#", ""})...)
	if IdentityCard != "123456789123456789" {
		ans = append(ans, AddStringToString(onlyFirst, IdentityCard[8:14], []string{"@", "_", "#", ""})...)
		ans = append(ans, AddStringToString(onlyFirst, IdentityCard[10:14], []string{"@", "_", "#", ""})...)
		ans = append(ans, AddStringToString(onlyFirst, IdentityCard[6:10], []string{"@", "_", "#", ""})...)
		ans = append(ans, AddStringToString(onlyFirst, IdentityCard[12:18], []string{"@", "_", "#", ""})...)
	}
	return ans
}

func GenerateWeakPassword(KeywordListStr string) []string {
	var KeywordList = []string{}
	if strings.Contains(KeywordListStr, ",") {
		KeywordList = strings.Split(KeywordListStr, ",")
	} else {
		KeywordList = append(KeywordList, KeywordListStr)
	}
	var PasswordList = []string{}
	for _, keyword := range KeywordList {
		for _, user := range common.Passwords {
			//如果是身份证格式
			if MightBeIdentityCard(keyword) {
				// 2009
				PasswordList = append(PasswordList, strings.ReplaceAll(user, "{user}", keyword[6:10]))
				// 后六位
				PasswordList = append(PasswordList, strings.ReplaceAll(user, "{user}", keyword[12:18]))
				// 年份后两位 + 生日
				PasswordList = append(PasswordList, strings.ReplaceAll(user, "{user}", keyword[8:14]))
				// 生日
				PasswordList = append(PasswordList, strings.ReplaceAll(user, "{user}", keyword[10:14]))
			} else if MightBeChineseName(keyword) {
				onlyFirst, firstComplete := TranslateToEnglish(keyword)
				PasswordList = append(PasswordList, strings.ReplaceAll(user, "{user}", onlyFirst))
				PasswordList = append(PasswordList, strings.ReplaceAll(user, "{user}", FirstCharToUpper(onlyFirst)))
				PasswordList = append(PasswordList, strings.ReplaceAll(user, "{user}", LastCharToUpper(onlyFirst)))
				PasswordList = append(PasswordList, strings.ReplaceAll(user, "{user}", strings.ToUpper(onlyFirst)))

				PasswordList = append(PasswordList, strings.ReplaceAll(user, "{user}", firstComplete))
				PasswordList = append(PasswordList, strings.ReplaceAll(user, "{user}", FirstCharToUpper(firstComplete)))
				PasswordList = append(PasswordList, strings.ReplaceAll(user, "{user}", LastCharToUpper(firstComplete)))
				PasswordList = append(PasswordList, strings.ReplaceAll(user, "{user}", strings.ToUpper(firstComplete)))
			} else {
				PasswordList = append(PasswordList, strings.ReplaceAll(user, "{user}", keyword))
				PasswordList = append(PasswordList, strings.ReplaceAll(user, "{user}", FirstCharToUpper(keyword)))
				PasswordList = append(PasswordList, strings.ReplaceAll(user, "{user}", LastCharToUpper(keyword)))
				PasswordList = append(PasswordList, strings.ReplaceAll(user, "{user}", strings.ToUpper(keyword)))
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
