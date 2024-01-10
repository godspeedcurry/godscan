package cmd

import (
	"fmt"
	"os"
	"strings"
	"sync"

	"github.com/fatih/color"
	"github.com/godspeedcurry/godscan/common"
	"github.com/godspeedcurry/godscan/utils"
	"github.com/gosuri/uiprogress"
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
	uiprogress.Start()
	bar := uiprogress.AddBar(len(targetUrlList)).AppendCompleted().PrependElapsed()

	for _, line := range targetUrlList {
		wg.Add(1)
		go func(url string) {
			defer wg.Done()
			result = append(result, utils.DirBrute(url, common.DirList)...)
			bar.Incr()
		}(line)
	}
	wg.Wait()
	uiprogress.Stop()
	utils.Success(color.GreenString("\n" + strings.Join(result, "\n")))

}

func init() {
	dirbruteCmd := newCommandWithAliases("dirbrute", "dirbrute on sensitive file", []string{"dir"}, &dirbruteOptions)
	rootCmd.AddCommand(dirbruteCmd)
}
