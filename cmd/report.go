package cmd

import (
	"database/sql"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/godspeedcurry/godscan/utils"
	prettytable "github.com/jedib0t/go-pretty/v6/table"
	"github.com/jedib0t/go-pretty/v6/text"
	"github.com/spf13/cobra"
)

var reportCmd = &cobra.Command{
	Use:   "report",
	Short: "View spider.db content or export HTML report",
	Run: func(cmd *cobra.Command, args []string) {
		htmlPath, _ := cmd.Flags().GetString("html")
		db, err := utils.InitSpiderDB("spider.db")
		if err != nil {
			utils.Error("open spider.db failed: %v", err)
			return
		}
		defer db.Close()
		printSummary(db)
		printAPICounts(db)
		printSensitiveCounts(db)
		if htmlPath == "" {
			now := time.Now()
			htmlPath = fmt.Sprintf("output/report-%04d-%02d-%02d.html", now.Year(), now.Month(), now.Day())
		}
		if err := utils.ExportHTMLReport(db, htmlPath); err != nil {
			utils.Error("export html failed: %v", err)
		} else {
			utils.Success("html exported to %s", htmlPath)
		}
	},
}

func init() {
	rootCmd.AddCommand(reportCmd)
	reportCmd.Flags().String("html", "", "Export spider.db to a standalone HTML report (default output/report-YYYY-MM-DD.html)")
}

func printSummary(db *sql.DB) {
	recs, err := utils.LoadSpiderSummaries(db)
	if err != nil {
		utils.Error("load spider_summary failed: %v", err)
		return
	}
	table := prettytable.NewWriter()
	table.SetOutputMirror(os.Stdout)
	table.AppendHeader(prettytable.Row{"Url", "IconHash (fofa/hunter)", "API Count", "URLs Found", "CDN URLs", "CDN Hosts", "Status", "Save Dir"})
	table.SetStyle(prettytable.StyleRounded)
	table.SetColumnConfigs([]prettytable.ColumnConfig{
		{Number: 6, Transformer: func(val interface{}) string {
			s := strings.TrimSpace(fmt.Sprint(val))
			if s == "" {
				return s
			}
			return text.FgYellow.Sprintf("%s", s)
		}},
	})
	for _, r := range recs {
		icon := r.IconHash
		if icon == "" {
			icon = "-"
		}
		table.AppendRow(prettytable.Row{r.Url, icon, r.ApiCount, r.UrlCount, r.CDNCount, r.CDNHosts, r.Status, r.SaveDir})
	}
	table.Render()
}

func printAPICounts(db *sql.DB) {
	type row struct {
		Root string
		Cnt  int
	}
	rows, err := utils.LoadAPICounts(db)
	if err != nil {
		utils.Error("load api_paths failed: %v", err)
		return
	}
	table := prettytable.NewWriter()
	table.SetOutputMirror(os.Stdout)
	table.AppendHeader(prettytable.Row{"Root URL", "API Paths"})
	table.SetStyle(prettytable.StyleRounded)
	for _, r := range rows {
		table.AppendRow(prettytable.Row{r.Root, r.Cnt})
	}
	table.Render()
}

func printSensitiveCounts(db *sql.DB) {
	rows, err := utils.LoadSensitiveCounts(db)
	if err != nil {
		utils.Error("load sensitive_hits failed: %v", err)
		return
	}
	table := prettytable.NewWriter()
	table.SetOutputMirror(os.Stdout)
	table.AppendHeader(prettytable.Row{"Category", "Hits"})
	table.SetStyle(prettytable.StyleRounded)
	for _, r := range rows {
		table.AppendRow(prettytable.Row{r.Category, r.Count})
	}
	table.Render()
}
