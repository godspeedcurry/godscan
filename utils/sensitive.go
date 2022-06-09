package utils

import (
	"regexp"
	"strings"

	"github.com/fatih/color"
)

func SensitiveInfoCollect(Content string) {
	color.HiMagenta("[*] Sensitive information")
	infoMap := map[string]string{
		"Chinese Mobile Number": `[^\w]((?:(?:\+|00)86)?1(?:(?:3[\d])|(?:4[5-79])|(?:5[0-35-9])|(?:6[5-7])|(?:7[0-8])|(?:8[\d])|(?:9[189]))\d{8})[^\w]`,
		"Email":                 `[0-9a-z][_.0-9a-z-]{0,31}@([0-9a-z][0-9a-z-]{0,30}[0-9a-z]\.){1,4}[a-z]{2,4}`,
		"HTML Notes":            `(<!--.*?-->)`,
		"Internal IP Address":   `[^0-9]((127\.0\.0\.1)|(10\.\d{1,3}\.\d{1,3}\.\d{1,3})|(172\.((1[6-9])|(2\d)|(3[01]))\.\d{1,3}\.\d{1,3})|(192\.168\.\d{1,3}\.\d{1,3}))`,
		"JSON Web Token":        `(eyJ[A-Za-z0-9_-]{10,}\.[A-Za-z0-9._-]{10,}|eyJ[A-Za-z0-9_\/+-]{10,}\.[A-Za-z0-9._\/+-]{10,})`,
		"Swagger UI":            `((swagger-ui.html)|(\"swagger\":)|(Swagger UI)|(swaggerUi))`,
		"Ueditor":               `(ueditor\.(config|all)\.js)`,
		"Windows File/Dir Path": `[^\w](([a-zA-Z]:\\(?:\w+\\?)*)|([a-zA-Z]:\\(?:\w+\\)*\w+\.\w+))`,
	}
	for key := range infoMap {
		reg := regexp.MustCompile(infoMap[key])
		res := reg.FindAllString(Content, -1)
		if len(res) > 0 {
			color.HiYellow("[*] %s\n", key)
			color.HiMagenta(strings.Join(res, "\n"))
		}
	}
}
