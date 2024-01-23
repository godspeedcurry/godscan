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
}

var (
	dirbruteOptions DirbruteOptions
	result          []string
	targetUrlList   []string
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
	utils.Info("Total: %d url(s)", len(targetUrlList))

	var wg sync.WaitGroup
	bar := pb.StartNew(len(targetUrlList) * len(common.DirList))

	table := tablewriter.NewWriter(os.Stdout)
	table.SetHeader([]string{"Url", "Title", "Finger", "Content-Type", "StatusCode", "Length"})

	for _, line := range targetUrlList {
		for _, dir := range common.DirList {
			wg.Add(1)
			go func(url string, dir string) {
				defer wg.Done()
				ret := utils.DirBrute(url, dir)
				if len(ret) != 0 {
					table.Append(ret)
				}
				bar.Increment()
			}(line, dir)
		}
	}
	wg.Wait()
	bar.Finish()
	table.Render()

}

func init() {
	dirbruteCmd := newCommandWithAliases("dirbrute", "Dirbrute on sensitive file", []string{"dir", "dirb", "dd"}, &dirbruteOptions)
	rootCmd.AddCommand(dirbruteCmd)
}
