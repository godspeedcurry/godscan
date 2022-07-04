package main

import (
	"main/common"
	"main/utils"
)

var Info common.HostInfo

func main() {
	common.Flag(&Info)
	if Info.Url != "" {
		utils.PrintFinger(Info)
	}
	if Info.Keywords != "" {
		utils.GeneateWeakPassword(Info.Keywords)
	}
}
