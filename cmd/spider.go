/*
Copyright © 2023 NAME HERE <EMAIL ADDRESS>
*/
package cmd

import (
	"fmt"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"database/sql"

	"github.com/cheggaaa/pb/v3"
	"github.com/godspeedcurry/godscan/common"
	"github.com/godspeedcurry/godscan/utils"
	prettytable "github.com/jedib0t/go-pretty/v6/table"
	"github.com/jedib0t/go-pretty/v6/text"
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
	start := time.Now()
	utils.InitHttp()
	targetUrlList := GetTargetList()
	utils.Info("Total: %d url(s)", len(targetUrlList))

	var wg sync.WaitGroup
	maxGoroutines := viper.GetInt("spider-threads")
	if maxGoroutines <= 0 {
		maxGoroutines = 20
	}
	sem := make(chan struct{}, maxGoroutines)
	results := make(chan utils.SpiderSummary, len(targetUrlList))
	var summaries []utils.SpiderSummary
	var mu sync.Mutex

	db, err := utils.InitSpiderDB("spider.db")
	if err != nil {
		utils.Error("failed to init spider.db: %v", err)
		return
	}
	utils.SetSpiderDB(db)
	defer db.Close()

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, os.Interrupt)
	defer signal.Stop(sigCh)
	go func() {
		for range sigCh {
			utils.Warning("Interrupted: data persisted to spider.db (run `godscan report` to view)")
			autoExportReport(db)
			os.Exit(1)
		}
	}()
	for _, line := range targetUrlList {
		sem <- struct{}{}
		wg.Add(1)
		go func(url string) {
			defer wg.Done()
			defer func() { <-sem }()
			res := utils.FingerSummary(url, o.Depth, db)
			results <- res
		}(line)
	}
	wg.Wait()
	close(results)

	table := prettytable.NewWriter()
	table.SetOutputMirror(os.Stdout)
	table.AppendHeader(prettytable.Row{"Url", "IconHash (fofa/hunter)", "API Count", "CDN URLs", "CDN Hosts"})
	table.SetStyle(prettytable.StyleRounded)
	table.SetColumnConfigs([]prettytable.ColumnConfig{
		{Number: 1, WidthMax: 64},
		{Number: 2, WidthMax: 48},
		{Number: 3, WidthMax: 10},
		{Number: 4, WidthMax: 10},
		{Number: 5, WidthMax: 36, Transformer: func(val interface{}) string {
			s := strings.TrimSpace(fmt.Sprint(val))
			if s == "" {
				return s
			}
			return text.FgYellow.Sprintf("%s", s)
		}},
	})

	total := len(targetUrlList)
	reachable := 0
	findings := 0
	var bar *pb.ProgressBar
	if !viper.GetBool("quiet") {
		bar = pb.StartNew(total)
		bar.SetMaxWidth(80)
		bar.SetTemplateString(`{{counters . }} {{bar . "[" "=" ">" " " "]"}} {{percent .}}`)
	}
	for res := range results {
		if bar != nil {
			bar.Increment()
		}
		if res.Status >= 0 {
			reachable++
		}
		if res.Finger != "" && res.Finger != common.NoFinger {
			findings++
		}
		if res.Status == -1 {
			continue
		}
		mu.Lock()
		summaries = append(summaries, res)
		if err := utils.SaveSpiderSummary(db, utils.SpiderRecord{
			Url:      res.URL,
			IconHash: res.IconHash,
			ApiCount: res.ApiCount,
			UrlCount: res.UrlCount,
			SaveDir:  res.SaveDir,
			Status:   res.Status,
		}); err != nil {
			utils.Error("db save failed: %v", err)
		}
		mu.Unlock()
		icon := res.IconHash
		if icon == "" {
			icon = "-"
		}
		row := []string{res.URL, icon, fmt.Sprintf("%d", res.ApiCount), fmt.Sprintf("%d", res.CDNCount), res.CDNHosts}
		utils.AddDataToTable(table, row)
	}
	if bar != nil {
		bar.Finish()
	}
	if table.Length() > 0 {
		table.Render()
	}
	utils.Info("Data persisted to spider.db (run `godscan report` to view)")
	autoExportReport(db)

	hostErrs := utils.CollectHostErrorStats(true)
	for host, stat := range hostErrs {
		utils.Warning("%s unreachable x%d: %s", host, stat.Count, stat.Sample)
	}
	utils.Info("Summary: %d urls | %d reachable | %d findings | %d host-errors | %s", total, reachable, findings, len(hostErrs), time.Since(start).Round(time.Millisecond))

}

func autoExportReport(db *sql.DB) {
	if db == nil {
		return
	}
	tmp := filepath.Join(".", "report.xlsx.tmp")
	final := filepath.Join(".", "report.xlsx")
	if err := exportXLSX(db, tmp); err != nil {
		utils.Error("auto-export xlsx failed: %v", err)
		return
	}
	if err := os.Rename(tmp, final); err != nil {
		utils.Error("rename report.xlsx failed: %v", err)
		return
	}
	utils.Success("report.xlsx updated")
}
