package cmd

import (
	"fmt"
	"os"

	"github.com/cheggaaa/pb/v3"
	"github.com/godspeedcurry/godscan/common"
	"github.com/godspeedcurry/godscan/utils"
	"github.com/olekukonko/tablewriter"
	"github.com/spf13/viper"
)

type DirbruteOptions struct {
	DirFile        string
	Threads        int
	FollowRedirect bool
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
	utils.Info("Total: %d threads", viper.GetInt("dirbrute-threads"))
	utils.Success("🌲🌲🌲 Log at ./dirbrute.csv")

	bar := pb.StartNew(len(targetUrlList) * len(targetDirList))

	table := tablewriter.NewWriter(os.Stdout)
	table.SetAutoWrapText(false)

	table.SetHeader(common.TableHeader)

	// 定义最大并发量
	maxGoroutines := viper.GetInt("dirbrute-threads")
	sem := make(chan struct{}, maxGoroutines)
	done := make(chan bool)

	go func() {
		for _, line := range targetUrlList {
			for _, dir := range targetDirList {
				sem <- struct{}{} // 向通道发送信号，表示一个新的协程即将启动

				go func(url string, dir string) {
					defer func() { <-sem }() // 从通道中取出信号，表示协程结束

					ret := utils.DirBrute(url, dir)
					utils.AddDataToTable(table, ret)
					bar.Increment()

					done <- true
				}(line, dir)
			}
		}
	}()

	// 等待所有任务完成
	for i := 0; i < len(targetUrlList)*len(targetDirList); i++ {
		<-done
	}
	bar.Finish()
	if table.NumLines() >= 1 {
		table.Render()
	}

}

func init() {
	dirbruteCmd := newCommandWithAliases("dirbrute", "Dirbrute on sensitive file", []string{"dir", "dirb", "dd"}, &dirbruteOptions)
	dirbruteCmd.PersistentFlags().StringVarP(&dirbruteOptions.DirFile, "dir-file", "", "", "your directory dict")

	dirbruteCmd.PersistentFlags().IntVarP(&dirbruteOptions.Threads, "threads", "t", 30, "Number of goroutine to use")

	dirbruteCmd.PersistentFlags().BoolVarP(&dirbruteOptions.FollowRedirect, "redirect", "L", false, "allow redirect")
	viper.BindPFlag("dirbrute-threads", dirbruteCmd.PersistentFlags().Lookup("threads"))
	viper.SetDefault("dirbrute-threads", 30)

	viper.BindPFlag("redirect", dirbruteCmd.PersistentFlags().Lookup("redirect"))
	viper.SetDefault("redirect", false)

	rootCmd.AddCommand(dirbruteCmd)
}
