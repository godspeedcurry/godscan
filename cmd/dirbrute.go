package cmd

import (
	"fmt"
	"os"
	"sync"

	"github.com/cheggaaa/pb/v3"
	"github.com/godspeedcurry/godscan/common"
	"github.com/godspeedcurry/godscan/utils"
	prettytable "github.com/jedib0t/go-pretty/v6/table"
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
	utils.Success("Log at ./dirbrute.csv")
	totalTasks := len(targetUrlList) * len(targetDirList)
	if totalTasks == 0 {
		utils.Warning("No tasks to run")
		return
	}

	bar := pb.StartNew(totalTasks)

	table := prettytable.NewWriter()
	table.SetOutputMirror(os.Stdout)
	table.AppendHeader(prettytable.Row(utils.StringListToInterfaceList(common.TableHeader)))
	table.SetStyle(prettytable.StyleRounded)

	maxGoroutines := viper.GetInt("dirbrute-threads")
	if maxGoroutines <= 0 {
		maxGoroutines = 1
	}
	tasks := make(chan struct {
		url string
		dir string
	})
	rows := make(chan []string, maxGoroutines)
	var workerWG sync.WaitGroup

	for i := 0; i < maxGoroutines; i++ {
		workerWG.Add(1)
		go func() {
			defer workerWG.Done()
			for task := range tasks {
				rows <- utils.DirBrute(task.url, task.dir)
			}
		}()
	}

	go func() {
		for _, line := range targetUrlList {
			for _, dir := range targetDirList {
				tasks <- struct {
					url string
					dir string
				}{url: line, dir: dir}
			}
		}
		close(tasks)
	}()

	go func() {
		workerWG.Wait()
		close(rows)
	}()

	for ret := range rows {
		if len(ret) > 0 {
			utils.AddDataToTable(table, ret)
		}
		bar.Increment()
	}
	bar.Finish()
	if table.Length() >= 1 {
		table.Render()
	}

}

func init() {
	dirbruteCmd := newCommandWithAliases("dirbrute", "Bruteforce common directories/files", []string{"dir", "dirb", "dd"}, &dirbruteOptions)
	dirbruteCmd.PersistentFlags().StringVarP(&dirbruteOptions.DirFile, "dir-file", "", "", "custom dictionary file")

	dirbruteCmd.PersistentFlags().IntVarP(&dirbruteOptions.Threads, "threads", "t", 30, "number of goroutines to use")

	dirbruteCmd.PersistentFlags().BoolVarP(&dirbruteOptions.FollowRedirect, "redirect", "L", false, "follow HTTP redirects")
	viper.BindPFlag("dirbrute-threads", dirbruteCmd.PersistentFlags().Lookup("threads"))
	viper.SetDefault("dirbrute-threads", 30)

	viper.BindPFlag("redirect", dirbruteCmd.PersistentFlags().Lookup("redirect"))
	viper.SetDefault("redirect", false)

	rootCmd.AddCommand(dirbruteCmd)
}
