package cmd

import (
	"github.com/godspeedcurry/godscan/utils"
)

func GetTargetList() []string {
	targetUrlList := []string{}
	if GlobalOption.Url != "" {
		targetUrlList = append(targetUrlList, GlobalOption.Url)
	} else {
		targetUrlList = append(targetUrlList, utils.FilReadUrl(GlobalOption.UrlFile)...)
	}
	return targetUrlList
}
