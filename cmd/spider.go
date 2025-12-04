/*
Copyright © 2023 NAME HERE <EMAIL ADDRESS>
*/
package cmd

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"sync"
	"sync/atomic"
	"time"

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
	Progress  bool
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
	spiderCmd.PersistentFlags().BoolVar(&spiderOptions.Progress, "progress-log", true, "print progress logs and per-target start notices")

	viper.BindPFlag("ApiPrefix", spiderCmd.PersistentFlags().Lookup("api"))
	viper.SetDefault("ApiPrefix", "")
	viper.BindPFlag("spider-threads", spiderCmd.PersistentFlags().Lookup("threads"))
	viper.SetDefault("spider-threads", 20)
	viper.BindPFlag("spider-progress-log", spiderCmd.PersistentFlags().Lookup("progress-log"))
	viper.SetDefault("spider-progress-log", true)

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
	progressLog := viper.GetBool("spider-progress-log")

	var wg sync.WaitGroup
	maxGoroutines := viper.GetInt("spider-threads")
	if maxGoroutines <= 0 {
		maxGoroutines = 20
	}
	sem := make(chan struct{}, maxGoroutines)
	results := make(chan utils.SpiderSummary, len(targetUrlList))
	var summaries []utils.SpiderSummary
	var mu sync.Mutex
	var processed int32
	var started int32
	var finished int32
	doneCh := make(chan struct{})

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
			if progressLog {
				curStart := atomic.AddInt32(&started, 1)
				remaining := int32(len(targetUrlList)) - curStart
				utils.Info("spider start: %s (started %d, remaining ~%d)", url, curStart, remaining)
			}
			res := utils.FingerSummary(url, o.Depth, db)
			if progressLog {
				curFinished := atomic.AddInt32(&finished, 1)
				remaining := int32(len(targetUrlList)) - curFinished
				utils.Info("spider finish: %s status=%d api=%d urls=%d | finished %d/%d remaining %d", res.URL, res.Status, res.ApiCount, res.UrlCount, curFinished, len(targetUrlList), remaining)
			}
			results <- res
		}(line)
	}
	// heartbeat to show progress even when no finishes yet
	if progressLog {
		go func() {
			ticker := time.NewTicker(5 * time.Second)
			defer ticker.Stop()
			for {
				select {
				case <-doneCh:
					return
				case <-ticker.C:
					curStart := atomic.LoadInt32(&started)
					curDone := atomic.LoadInt32(&processed)
					inflight := curStart - curDone
					if inflight < 0 {
						inflight = 0
					}
					utils.Info("spider heartbeat: started %d done %d inflight %d total %d", curStart, curDone, inflight, len(targetUrlList))
				}
			}
		}()
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
	progressMilestone := 10
	if !viper.GetBool("quiet") {
		bar = pb.StartNew(total)
		bar.SetMaxWidth(90)
		bar.Set("prefix", "spider")
		bar.SetTemplateString(`{{string . "prefix"}} {{counters .}} {{bar . "[" "=" ">" " " "]"}} {{percent .}} | elapsed: {{etime .}} | eta: {{rtime .}}`)
		bar.SetRefreshRate(200 * time.Millisecond)
	}
	for res := range results {
		if bar != nil {
			bar.Increment()
			cur := atomic.AddInt32(&processed, 1)
			bar.Set("prefix", fmt.Sprintf("spider %d/%d", cur, total))
		}
		cur := atomic.LoadInt32(&processed)
		if progressLog && total > 0 && progressMilestone <= 100 {
			percent := int(cur) * 100 / total
			if percent >= progressMilestone {
				utils.Info("spider progress: %d/%d (%d%%)", cur, total, percent)
				progressMilestone += 10
			}
		}
		if res.Status >= 0 {
			reachable++
		}
		if res.Finger != "" && res.Finger != common.NoFinger {
			findings++
		}
		if progressLog {
			successRate := float64(reachable) / float64(total)
			remaining := total - int(cur)
			utils.Info("spider finish: %s status=%d api=%d urls=%d | done %d/%d (%.1f%% success) remaining %d", res.URL, res.Status, res.ApiCount, res.UrlCount, cur, total, successRate*100, remaining)
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
	writeSpiderJSONSummary(summaries)
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

func writeSpiderJSONSummary(summaries []utils.SpiderSummary) {
	if len(summaries) == 0 {
		return
	}
	type out struct {
		URL         string   `json:"url"`
		Title       string   `json:"title"`
		Finger      string   `json:"finger"`
		ContentType string   `json:"content_type"`
		Status      int      `json:"status"`
		Length      int      `json:"length"`
		Keyword     string   `json:"keyword"`
		SimHash     string   `json:"simhash"`
		IconHash    string   `json:"icon_hash"`
		ApiCount    int      `json:"api_count"`
		UrlCount    int      `json:"url_count"`
		CDNCount    int      `json:"cdn_count"`
		CDNHosts    []string `json:"cdn_hosts"`
		SaveDir     string   `json:"save_dir"`
	}
	outList := make([]out, 0, len(summaries))
	for _, s := range summaries {
		cdns := []string{}
		if s.CDNHosts != "" {
			for _, h := range strings.Split(s.CDNHosts, ",") {
				h = strings.TrimSpace(h)
				if h != "" {
					cdns = append(cdns, h)
				}
			}
		}
		outList = append(outList, out{
			URL:         s.URL,
			Title:       s.Title,
			Finger:      s.Finger,
			ContentType: s.ContentType,
			Status:      s.Status,
			Length:      s.Length,
			Keyword:     s.Keyword,
			SimHash:     s.SimHash,
			IconHash:    s.IconHash,
			ApiCount:    s.ApiCount,
			UrlCount:    s.UrlCount,
			CDNCount:    s.CDNCount,
			CDNHosts:    cdns,
			SaveDir:     s.SaveDir,
		})
	}
	data, err := json.MarshalIndent(outList, "", "  ")
	if err != nil {
		utils.Debug("failed to marshal spider summary: %v", err)
		return
	}
	outDir := viper.GetString("output-dir")
	if outDir == "" {
		outDir = "."
	}
	outPath := filepath.Join(outDir, "spider_summary.json")
	if err := os.WriteFile(outPath, data, 0644); err != nil {
		utils.Debug("failed to write spider_summary.json: %v", err)
		return
	}
	utils.Info("spider_summary.json updated at %s", outPath)
}
