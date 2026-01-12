package cmd

import (
	"context"
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

var reportLLMOpts LLMCLIOptions

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

		llmCfg := promptLLMIfNeeded(cmd, &reportLLMOpts)
		if llmCfg != nil {
			utils.Info("LLM provider=%s model=%s", llmCfg.Provider, llmCfg.Model)
		}
		printSummary(db)
		printAPICounts(db)
		printSensitiveCounts(db)
		if htmlPath == "" {
			now := time.Now()
			htmlPath = fmt.Sprintf("output/report-%04d-%02d-%02d.html", now.Year(), now.Month(), now.Day())
		}
		ctx := context.Background()
		if err := utils.ExportHTMLReport(ctx, db, htmlPath, llmCfg); err != nil {
			utils.Error("export html failed: %v", err)
		} else {
			utils.Success("html exported to %s", htmlPath)
		}
	},
}

func init() {
	rootCmd.AddCommand(reportCmd)
	reportCmd.Flags().String("html", "", "Export spider.db to a standalone HTML report (default output/report-YYYY-MM-DD.html)")
	addLLMFlags(reportCmd, &reportLLMOpts)
}

func promptLLMIfNeeded(cmd *cobra.Command, opts *LLMCLIOptions) *utils.LLMConfig {
	if cmd.Flags().NFlag() > 0 || opts == nil || opts.Profile != "" || opts.DryRun {
		return opts.ToConfig()
	}
	if !stdinIsTTY() {
		return nil
	}
	if !promptYesNo("Enable LLM summary in report? [y/N]: ") {
		return nil
	}
	name := strings.TrimSpace(promptString("LLM profile name (secret): "))
	if name == "" {
		utils.Info("LLM skipped: empty profile name")
		return nil
	}
	opts.Profile = name
	if cfg := opts.ToConfig(); cfg != nil {
		return cfg
	}

	// No profile found or key missing; guide user to create one quickly.
	path := utils.DefaultLLMProfilePath("")
	profiles, _ := utils.LoadLLMProfiles(path, name) // ignore error; will recreate
	provider := strings.TrimSpace(promptDefault("Provider", utils.DefaultLLMProvider))
	if provider == "" {
		provider = utils.DefaultLLMProvider
	}
	model := strings.TrimSpace(promptDefault("Model", utils.DefaultLLMModel))
	if model == "" {
		model = utils.DefaultLLMModel
	}
	key := strings.TrimSpace(promptString("API key (required): "))
	if key == "" {
		utils.Warning("LLM skipped: API key empty")
		return nil
	}
	p := utils.LLMProfile{
		Name:     name,
		Provider: provider,
		Model:    model,
		APIKey:   key,
	}
	profiles = utils.UpsertProfile(profiles, p)
	if err := utils.SaveLLMProfiles(path, name, profiles); err != nil {
		utils.Warning("save profile failed: %v", err)
		return nil
	}
	utils.Success("LLM profile %s saved (%s)", name, path)
	return opts.ToConfig()
}

func promptYesNo(msg string) bool {
	fmt.Print(msg)
	var input string
	_, _ = fmt.Scanln(&input)
	input = strings.TrimSpace(strings.ToLower(input))
	return strings.HasPrefix(input, "y")
}

func promptString(msg string) string {
	fmt.Print(msg)
	var input string
	_, _ = fmt.Scanln(&input)
	return input
}

func promptDefault(label, def string) string {
	fmt.Printf("%s [%s]: ", label, def)
	var input string
	_, _ = fmt.Scanln(&input)
	input = strings.TrimSpace(input)
	if input == "" {
		return def
	}
	return input
}

func stdinIsTTY() bool {
	info, err := os.Stdin.Stat()
	if err != nil {
		return false
	}
	return (info.Mode() & os.ModeCharDevice) != 0
}

func printSummary(db *sql.DB) {
	recs, err := utils.LoadSpiderSummaries(db)
	if err != nil {
		utils.Error("load spider_summary failed: %v", err)
		return
	}
	table := prettytable.NewWriter()
	table.SetOutputMirror(os.Stdout)
	table.AppendHeader(prettytable.Row{"Url", "IconHash", "API Count", "URLs Found", "CDN URLs", "CDN Hosts", "Status", "Save Dir"})
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
