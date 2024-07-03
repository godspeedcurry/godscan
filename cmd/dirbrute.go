package cmd

import (
	"fmt"
	"os"
	"sync"

	"github.com/cheggaaa/pb/v3"
	"github.com/godspeedcurry/godscan/common"
	"github.com/godspeedcurry/godscan/utils"
	"github.com/olekukonko/tablewriter"
)

type DirbruteOptions struct {
	DirFile string
}

var (
	dirbruteOptions DirbruteOptions
)

func (o *DirbruteOptions) validateOptions() error {
	if GlobalOption.Url == "" && GlobalOption.UrlFile == "" {
		return fmt.Errorf("please give target url")
	}
	if GlobalOption.UrlFile != "" {
		_, err := os.Stat(GlobalOption.UrlFile)
		if err != nil {
			return fmt.Errorf("file not exist")
		}
	}
	return nil
}

func (o *DirbruteOptions) run() {
	utils.InitHttp()
	targetUrlList := GetTargetList()

	targetDirList := common.DirList
	if o.DirFile != "" {
		targetDirList = utils.FileReadLine(o.DirFile)
	}
	utils.Info("Total: %d url(s)", len(targetUrlList))
	utils.Info("Total: %d payload(s) in dir dict", len(targetDirList))
	utils.Success("🌲🌲🌲 Log at ./dirbrute.csv")
	var wg sync.WaitGroup
	bar := pb.StartNew(len(targetUrlList) * len(targetDirList))

	table := tablewriter.NewWriter(os.Stdout)
	table.SetAutoWrapText(false)

	table.SetHeader(common.TableHeader)

	for _, line := range targetUrlList {
		for _, dir := range targetDirList {
			wg.Add(1)
			go func(url string, dir string) {
				defer wg.Done()
				ret := utils.DirBrute(url, dir)
				utils.AddDataToTable(table, ret)
				bar.Increment()
			}(line, dir)
		}
	}
	wg.Wait()
	bar.Finish()
	if table.NumLines() >= 1 {
		table.Render()
	}

}

func init() {
	dirbruteCmd := newCommandWithAliases("dirbrute", "Dirbrute on sensitive file", []string{"dir", "dirb", "dd"}, &dirbruteOptions)
	dirbruteCmd.PersistentFlags().StringVarP(&dirbruteOptions.DirFile, "dir-file", "", "", "your directory dict")

	rootCmd.AddCommand(dirbruteCmd)
}
