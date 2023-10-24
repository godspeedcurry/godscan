/*
Copyright © 2023 NAME HERE <EMAIL ADDRESS>
*/
package cmd

import (
	"fmt"
	"sync"

	"github.com/godspeedcurry/godscan/utils"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

type SpiderOptions struct {
	Depth     int
	ApiPrefix string
}

var (
	spiderOptions SpiderOptions
)

// spiderCmd represents the spider command
var spiderCmd = &cobra.Command{
	Use:   "spider",
	Short: "analyze website using DFS",
	Run: func(cmd *cobra.Command, args []string) {
		if err := spiderOptions.validateOptions(); err != nil {
			fmt.Println("Try 'spider --url www.baidu.com")
			return
		}
		spiderOptions.run()
	},
}

func init() {
	rootCmd.AddCommand(spiderCmd)
	spiderCmd.PersistentFlags().IntVarP(&spiderOptions.Depth, "depth", "d", 1, "your search depth, default 1")
	spiderCmd.PersistentFlags().StringVarP(&spiderOptions.ApiPrefix, "api", "", "", "your api prefix")

	viper.BindPFlag("ApiPrefix", spiderCmd.PersistentFlags().Lookup("ApiPrefix"))
	viper.SetDefault("ApiPrefix", "")

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
	for _, line := range targetUrlList {
		wg.Add(1)
		go func(url string) {
			defer wg.Done()
			utils.PrintFinger(url, spiderOptions.Depth)
		}(line)
	}
	wg.Wait()

}
