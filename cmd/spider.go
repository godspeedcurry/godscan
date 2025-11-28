/*
Copyright © 2023 NAME HERE <EMAIL ADDRESS>
*/
package cmd

import (
	"fmt"
	"sync"

	"github.com/godspeedcurry/godscan/utils"
	"github.com/spf13/viper"
)

type SpiderOptions struct {
	Depth     int
	ApiPrefix string
	Threads   int
}

var (
	spiderOptions SpiderOptions
)

func init() {

	spiderCmd := newCommandWithAliases("spider", "Analyze website using DFS, quick usage: -u", []string{"sp", "ss"}, &spiderOptions)
	rootCmd.AddCommand(spiderCmd)
	spiderCmd.PersistentFlags().IntVarP(&spiderOptions.Depth, "depth", "d", 2, "your search depth, default 2")
	spiderCmd.PersistentFlags().StringVarP(&spiderOptions.ApiPrefix, "api", "", "", "your api prefix")
	spiderCmd.PersistentFlags().IntVarP(&spiderOptions.Threads, "threads", "t", 20, "Number of concurrent targets")

	viper.BindPFlag("ApiPrefix", spiderCmd.PersistentFlags().Lookup("api"))
	viper.SetDefault("ApiPrefix", "")
	viper.BindPFlag("spider-threads", spiderCmd.PersistentFlags().Lookup("threads"))
	viper.SetDefault("spider-threads", 20)

}

func (o *SpiderOptions) validateOptions() error {
	if GlobalOption.Url == "" && GlobalOption.UrlFile == "" {
		return fmt.Errorf("please give target url")
	}
	return nil
}

func (o *SpiderOptions) run() {
	utils.InitHttp()
	targetUrlList := GetTargetList()
	utils.Info("Total: %d url(s)", len(targetUrlList))

	var wg sync.WaitGroup
	maxGoroutines := viper.GetInt("spider-threads")
	if maxGoroutines <= 0 {
		maxGoroutines = 20
	}
	sem := make(chan struct{}, maxGoroutines)
	for _, line := range targetUrlList {
		sem <- struct{}{}
		wg.Add(1)
		go func(url string) {
			defer wg.Done()
			defer func() { <-sem }()
			utils.PrintFinger(url, o.Depth)
		}(line)
	}
	wg.Wait()

}
