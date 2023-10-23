package utils

import (
	"html"
	"regexp"

	"github.com/fatih/color"
)

func SensitiveInfoCollect(Url string, Content string) {
	infoMap := map[string]string{
		"Chinese Mobile Number": `[^\d]((?:(?:\+|00)86)?1(?:(?:3[\d])|(?:4[5-79])|(?:5[0-35-9])|(?:6[5-7])|(?:7[0-8])|(?:8[\d])|(?:9[189]))\d{8})[^\d]`,
		"Email":                 `([0-9a-z][_.0-9a-z-]{0,31}@([0-9a-z][0-9a-z-]{0,30}[0-9a-z]\.){1,4}(com|net|cn))`,
		"Internal IP Address":   `[^0-9]((127\.0\.0\.1)|(10\.\d{1,3}\.\d{1,3}\.\d{1,3})|(172\.((1[6-9])|(2\d)|(3[01]))\.\d{1,3}\.\d{1,3})|(192\.168\.\d{1,3}\.\d{1,3}))`,
		"JSON Web Token":        `(eyJ[A-Za-z0-9_-]{10,}\.[A-Za-z0-9._-]{10,}|eyJ[A-Za-z0-9_\/+-]{10,}\.[A-Za-z0-9._\/+-]{10,})`,
		"Swagger UI":            `((swagger-ui.html)|(\"swagger\":)|(Swagger UI)|(swaggerUi))`,
		"Ueditor":               `(ueditor\.(config|all)\.js)`,
		"Windows File/Dir Path": `([a-fA-FzZ]:(\\{1,2})([^\n ]*\\?)*)`,
		"accesskey/accessid":    `(?i)(access([_ ]?(key|id|secret)){1,2}[\&\#0-9\" =;]*?([0-9a-zA-Z]{10,64}))`,
	}
	for key := range infoMap {
		reg := regexp.MustCompile(infoMap[key])
		res := reg.FindAllStringSubmatch(html.UnescapeString(Content), -1)
		if len(res) > 0 {
			color.HiYellow("->[*] %s\n", key)
			mylist := []string{}
			for _, tmp := range res {
				r := regexp.MustCompile("png|gif|svg|jpeg|jpg|otf|woff|eot|ttf")
				if key == "link" && r.MatchString(tmp[1]) {
					continue
				}
				mylist = append(mylist, tmp[1])
			}
			unDupList := removeDuplicatesString(mylist)
			for _, a := range unDupList {
				color.HiMagenta(Url + " " + a)
			}
		}
	}
}
