package cmd

import (
	"database/sql"
	"fmt"
	"os"
	"strings"

	"github.com/godspeedcurry/godscan/utils"
	prettytable "github.com/jedib0t/go-pretty/v6/table"
	"github.com/jedib0t/go-pretty/v6/text"
	"github.com/spf13/cobra"
)

var reportCmd = &cobra.Command{
	Use:   "report",
	Short: "View spider.db content in tables",
	Run: func(cmd *cobra.Command, args []string) {
		xlsxPath, _ := cmd.Flags().GetString("xlsx")
		db, err := utils.InitSpiderDB("spider.db")
		if err != nil {
			utils.Error("open spider.db failed: %v", err)
			return
		}
		defer db.Close()
		printSummary(db)
		printAPICounts(db)
		printSensitiveCounts(db)
		if xlsxPath != "" {
			if err := exportXLSX(db, xlsxPath); err != nil {
				utils.Error("export xlsx failed: %v", err)
			} else {
				utils.Success("xlsx exported to %s", xlsxPath)
			}
		}
	},
}

func init() {
	rootCmd.AddCommand(reportCmd)
	reportCmd.Flags().String("xlsx", "", "Export spider.db to an xlsx file")
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

func exportXLSX(db *sql.DB, path string) error {
	summary, err := utils.LoadSpiderSummaries(db)
	if err != nil {
		return err
	}
	apiCounts, err := utils.LoadAPICounts(db)
	if err != nil {
		return err
	}
	sensCounts, err := utils.LoadSensitiveCounts(db)
	if err != nil {
		return err
	}
	sensHits, err := utils.LoadSensitiveHits(db)
	if err != nil {
		return err
	}
	var sheets []utils.Worksheet

	// summary sheet
	sumRows := [][]string{{"Url", "IconHash (fofa/hunter)", "API Count", "URLs Found", "CDN URLs", "CDN Hosts", "Status", "Save Dir"}}
	for _, r := range summary {
		icon := r.IconHash
		if icon == "" {
			icon = "-"
		}
		sumRows = append(sumRows, []string{
			r.Url,
			icon,
			fmt.Sprintf("%d", r.ApiCount),
			fmt.Sprintf("%d", r.UrlCount),
			fmt.Sprintf("%d", r.CDNCount),
			r.CDNHosts,
			fmt.Sprintf("%d", r.Status),
			r.SaveDir,
		})
	}
	sheets = append(sheets, utils.Worksheet{Name: "summary", Rows: sumRows})

	// api sheet
	apiRows := [][]string{{"Root URL", "API Paths"}}
	for _, a := range apiCounts {
		apiRows = append(apiRows, []string{a.Root, fmt.Sprintf("%d", a.Cnt)})
	}
	sheets = append(sheets, utils.Worksheet{Name: "api_paths", Rows: apiRows})

	// sensitive sheet
	sensRows := [][]string{{"Category", "Hits"}}
	for _, s := range sensCounts {
		sensRows = append(sensRows, []string{s.Category, fmt.Sprintf("%d", s.Count)})
	}
	sheets = append(sheets, utils.Worksheet{Name: "sensitive_hits", Rows: sensRows})

	// entropy sheet
	entHits, err := utils.LoadEntropyHits(db)
	if err != nil {
		return err
	}
	entRows := [][]string{{"Source URL", "Category", "Content", "Entropy", "Save Dir"}}
	for _, e := range entHits {
		entRows = append(entRows, []string{e.SourceURL, e.Category, e.Content, fmt.Sprintf("%.2f", e.Entropy), e.SaveDir})
	}
	sheets = append(sheets, utils.Worksheet{Name: "entropy_hits", Rows: entRows})

	// sensitive hits detail
	sensDetail := [][]string{{"Source URL", "Category", "Content", "Entropy", "Save Dir"}}
	for _, s := range sensHits {
		sensDetail = append(sensDetail, []string{s.SourceURL, s.Category, s.Content, fmt.Sprintf("%.2f", s.Entropy), s.SaveDir})
	}
	sheets = append(sheets, utils.Worksheet{Name: "sensitive_detail", Rows: sensDetail})

	// cdn hosts sheet
	cdns, err := utils.LoadCDNHosts(db)
	if err != nil {
		return err
	}
	cdnRows := [][]string{{"Root URL", "CDN Host"}}
	for _, c := range cdns {
		cdnRows = append(cdnRows, []string{c.Root, c.Host})
	}
	sheets = append(sheets, utils.Worksheet{Name: "cdn_hosts", Rows: cdnRows})

	return utils.WriteSimpleXLSX(path, sheets)
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
