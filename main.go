package main

import (
	"main/common"
	"main/utils"
)

var Info common.HostInfo

func main() {
	common.Flag(&Info)
	if Info.Url != "" {
		if Info.DirBrute {
			utils.DirSingleBrute(Info.Url)
		} else {
			utils.PrintFinger(Info)
		}
	}
	if Info.IconUrl != "" {
		utils.IconDetect(Info.IconUrl)
	}
	if Info.Keywords != "" {
		utils.GenerateWeakPassword(Info.Keywords, Info.Suffix)
	}

	if Info.UrlFile != "" {
		utils.DirBrute(Info.UrlFile)
	}
}
