package main

import (
	"main/common"
	"main/utils"
)

var Info common.HostInfo

func main() {
	common.Flag(&Info)
	if Info.Show {
		utils.ShowInfo()
		return
	}
	if Info.Url != "" {
		if Info.DirBrute {
			utils.DirSingleBrute(Info.Url)
		} else {
			utils.PrintFinger(Info)
		}
		return
	}
	if Info.IconUrl != "" {
		utils.IconDetect(Info.IconUrl)
		return
	}
	if Info.Keywords != "" {
		utils.GenerateWeakPassword(Info)
		return
	}

	if Info.UrlFile != "" {
		utils.DirBrute(Info.UrlFile)
		return
	}
}
