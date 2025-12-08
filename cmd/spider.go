/*
Copyright © 2023 NAME HERE <EMAIL ADDRESS>
*/
package cmd

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"os"
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

type spiderContext struct {
	started   int32
	processed int32
	finished  int32
	doneCh    chan struct{}
}

func spawnSpiderWorkers(targets []string, depth int, db *sql.DB, progressLog bool) (*sync.WaitGroup, chan utils.SpiderSummary, *spiderContext) {
	var wg sync.WaitGroup
	sem := make(chan struct{}, maxSpiderGoroutines())
	results := make(chan utils.SpiderSummary, len(targets))
	ctx := &spiderContext{doneCh: make(chan struct{})}

	for _, line := range targets {
		sem <- struct{}{}
		wg.Add(1)
		go func(url string) {
			defer wg.Done()
			defer func() { <-sem }()
			if progressLog {
				curStart := atomic.AddInt32(&ctx.started, 1)
				remaining := int32(len(targets)) - curStart
				utils.Info("spider start: %s (started %d, remaining ~%d)", url, curStart, remaining)
			}
			res := utils.FingerSummary(url, depth, db)
			if progressLog {
				curFinished := atomic.AddInt32(&ctx.finished, 1)
				remaining := int32(len(targets)) - curFinished
				utils.Info("spider finish: %s status=%d api=%d urls=%d | finished %d/%d remaining %d", res.URL, res.Status, res.ApiCount, res.UrlCount, curFinished, len(targets), remaining)
			}
			results <- res
		}(line)
	}
	return &wg, results, ctx
}

func maxSpiderGoroutines() int {
	maxGoroutines := viper.GetInt("spider-threads")
	if maxGoroutines <= 0 {
		maxGoroutines = 20
	}
	return maxGoroutines
}

func heartbeat(progressLog bool, ctx *spiderContext, total int) {
	if !progressLog {
		return
	}
	go func() {
		ticker := time.NewTicker(5 * time.Second)
		defer ticker.Stop()
		for {
			select {
			case <-ctx.doneCh:
				return
			case <-ticker.C:
				curStart := atomic.LoadInt32(&ctx.started)
				curDone := atomic.LoadInt32(&ctx.processed)
				inflight := curStart - curDone
				if inflight < 0 {
					inflight = 0
				}
				utils.Info("spider heartbeat: started %d done %d inflight %d total %d", curStart, curDone, inflight, total)
			}
		}
	}()
}

func collectSpiderResults(results chan utils.SpiderSummary, total int, progressLog bool) (prettytable.Writer, []utils.SpiderSummary, int, int) {
	collector := newSpiderCollector(total, progressLog)
	for res := range results {
		collector.updateProgress()
		collector.processResult(res)
	}
	collector.finish()
	return collector.table, collector.summaries, collector.reachable, collector.findings
}

func renderSpiderTable(table prettytable.Writer) {
	if table.Length() > 0 {
		table.Render()
	}
}

type spiderCollector struct {
	table             prettytable.Writer
	summaries         []utils.SpiderSummary
	reachable         int
	findings          int
	bar               *pb.ProgressBar
	processed         int32
	progressLog       bool
	progressMilestone int
	total             int
	mu                sync.Mutex
}

func cdnHostColor(val interface{}) string {
	s := strings.TrimSpace(fmt.Sprint(val))
	if s == "" || s == "-" {
		return s
	}
	return text.FgYellow.Sprintf("%s", s)
}

func newSpiderCollector(total int, progressLog bool) *spiderCollector {
	table := prettytable.NewWriter()
	table.SetOutputMirror(os.Stdout)
	table.AppendHeader(prettytable.Row{"Url", "IconHash (fofa/hunter)", "API Count", "CDN URLs", "CDN Hosts"})
	table.SetStyle(prettytable.StyleRounded)
	table.SetColumnConfigs([]prettytable.ColumnConfig{
		{Number: 1, WidthMax: 64},
		{Number: 2, WidthMax: 48},
		{Number: 3, WidthMax: 10},
		{Number: 4, WidthMax: 10},
		{Number: 5, WidthMax: 36, Transformer: cdnHostColor},
	})

	var bar *pb.ProgressBar
	if !viper.GetBool("quiet") {
		bar = pb.StartNew(total)
		bar.SetMaxWidth(90)
		bar.Set("prefix", "spider")
		bar.SetTemplateString(`{{string . "prefix"}} {{counters .}} {{bar . "[" "=" ">" " " "]"}} {{percent .}} | elapsed: {{etime .}} | eta: {{rtime .}}`)
		bar.SetRefreshRate(200 * time.Millisecond)
	}

	return &spiderCollector{
		table:             table,
		bar:               bar,
		progressLog:       progressLog,
		progressMilestone: 10,
		total:             total,
	}
}

func (c *spiderCollector) updateProgress() {
	if c.bar != nil {
		c.bar.Increment()
		cur := atomic.AddInt32(&c.processed, 1)
		c.bar.Set("prefix", fmt.Sprintf("spider %d/%d", cur, c.total))
		if !c.progressLog || c.total == 0 {
			return
		}
		percent := int(cur) * 100 / c.total
		if percent >= c.progressMilestone {
			utils.Info("spider progress: %d/%d (%d%%)", cur, c.total, percent)
			c.progressMilestone += 10
		}
	}
}

func (c *spiderCollector) processResult(res utils.SpiderSummary) {
	if res.Status >= 0 {
		c.reachable++
	}
	if res.Finger != "" && res.Finger != common.NoFinger {
		c.findings++
	}
	if res.Status == -1 {
		return
	}

	c.mu.Lock()
	c.summaries = append(c.summaries, res)
	if err := utils.SaveSpiderSummary(utils.GetSpiderDB(), utils.SpiderRecord{
		Url:      res.URL,
		IconHash: res.IconHash,
		ApiCount: res.ApiCount,
		UrlCount: res.UrlCount,
		SaveDir:  res.SaveDir,
		Status:   res.Status,
	}); err != nil {
		utils.Error("db save failed: %v", err)
	}
	c.mu.Unlock()

	icon := res.IconHash
	if icon == "" {
		icon = "-"
	}
	cdnHosts := res.CDNHosts
	if cdnHosts == "" {
		cdnHosts = "-"
	}
	row := []string{res.URL, icon, fmt.Sprintf("%d", res.ApiCount), fmt.Sprintf("%d", res.CDNCount), cdnHosts}
	utils.AddDataToTable(c.table, row)
}

func (c *spiderCollector) finish() {
	if c.bar != nil {
		c.bar.Finish()
	}
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
	targets := GetTargetList()
	utils.Info("Total: %d url(s)", len(targets))

	db, err := utils.InitSpiderDB("spider.db")
	if err != nil {
		utils.Error("failed to init spider.db: %v", err)
		return
	}
	utils.SetSpiderDB(db)
	defer db.Close()

	progressLog := viper.GetBool("spider-progress-log")
	wg, results, ctx := spawnSpiderWorkers(targets, o.Depth, db, progressLog)
	heartbeat(progressLog, ctx, len(targets))
	wg.Wait()
	close(results)

	table, summaries, reachable, findings := collectSpiderResults(results, len(targets), progressLog)
	close(ctx.doneCh)
	renderSpiderTable(table)
	writeSpiderJSONSummary(summaries)
	utils.Info("Data persisted to spider.db (run `godscan report` to view)")
	autoExportReport(db)

	hostErrs := utils.CollectHostErrorStats(true)
	for host, stat := range hostErrs {
		utils.Warning("%s unreachable x%d: %s", host, stat.Count, stat.Sample)
	}
	utils.Info("Summary: %d urls | %d reachable | %d findings | %d host-errors | %s", len(targets), reachable, findings, len(hostErrs), time.Since(start).Round(time.Millisecond))

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
