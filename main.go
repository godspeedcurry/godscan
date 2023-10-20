package main

import (
	"github.com/godspeedcurry/godscan/common"
	"github.com/godspeedcurry/godscan/utils"
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

// package main

// import "github.com/godspeedcurry/godscan/cmd"

// func main() {
// 	cmd.Execute()
// }
