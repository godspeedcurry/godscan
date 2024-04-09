package cmd

import (
	"fmt"
	"io"
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
	file, err := os.OpenFile("dirbrute.log", os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0644)
	if err != nil {
		fmt.Println("Failed to open file:", err)
		return
	}
	defer file.Close()

	multiWriter := io.MultiWriter(os.Stdout, file)
	table := tablewriter.NewWriter(multiWriter)
	table.SetAutoWrapText(false)

	table.SetHeader([]string{"Url", "Title", "Finger", "Content-Type", "StatusCode", "Length"})

	for _, line := range targetUrlList {
		for _, dir := range common.DirList {
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
	rootCmd.AddCommand(dirbruteCmd)
}
