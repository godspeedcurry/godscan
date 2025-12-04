package cmd

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"

	"github.com/godspeedcurry/godscan/utils"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

type SearchOptions struct {
	Pattern    string
	UrlLike    string
	MaxResult  int
	DbPath     string
	IgnoreCase bool
	Tables     []string
}

var searchOptions SearchOptions

func init() {
	searchCmd := &cobra.Command{
		Use:     "search [pattern]",
		Aliases: []string{"grep"},
		Short:   "Regex search in spider.db (api_paths & sensitive_hits)",
		Run: func(cmd *cobra.Command, args []string) {
			if searchOptions.Pattern == "" && len(args) > 0 {
				searchOptions.Pattern = args[0]
			}
			if err := runSearch(searchOptions); err != nil {
				utils.Error("%v", err)
			}
		},
	}
	searchCmd.Flags().StringVar(&searchOptions.Pattern, "pattern", "", "regex pattern to search (can also pass as first arg)")
	searchCmd.Flags().StringVar(&searchOptions.UrlLike, "url-like", "", "optional substring to filter source/root url")
	searchCmd.Flags().IntVar(&searchOptions.MaxResult, "limit", 200, "max results to display")
	searchCmd.Flags().StringVar(&searchOptions.DbPath, "db", "spider.db", "path to spider.db")
	searchCmd.Flags().BoolVarP(&searchOptions.IgnoreCase, "ignore-case", "i", false, "case-insensitive regex")
	searchCmd.Flags().StringSliceVar(&searchOptions.Tables, "table", []string{"api", "sensitive", "map"}, "tables to search: api,sensitive,map")
	rootCmd.AddCommand(searchCmd)
}

func runSearch(opt SearchOptions) error {
	if opt.Pattern == "" {
		return fmt.Errorf("pattern is required")
	}
	if _, err := os.Stat(opt.DbPath); err != nil {
		return fmt.Errorf("db not found: %s", opt.DbPath)
	}
	db, err := utils.InitSpiderDB(opt.DbPath)
	if err != nil {
		return fmt.Errorf("open db failed: %w", err)
	}
	defer db.Close()

	flags := ""
	if opt.IgnoreCase {
		flags = "(?i)"
	}
	re, err := regexp.Compile(flags + opt.Pattern)
	if err != nil {
		return fmt.Errorf("bad regex: %w", err)
	}

	searchAPI := false
	searchSensitive := false
	searchMaps := false
	for _, t := range opt.Tables {
		t = strings.ToLower(strings.TrimSpace(t))
		if t == "api" {
			searchAPI = true
		}
		if t == "sensitive" {
			searchSensitive = true
		}
		if t == "map" || t == "sourcemap" || t == "sourcemaps" {
			searchMaps = true
		}
	}
	if !searchAPI && !searchSensitive && !searchMaps {
		searchAPI, searchSensitive, searchMaps = true, true, true
	}

	apiRows := []apiRow{}
	if searchAPI {
		apiRows, err = queryAPIPaths(db, opt.UrlLike)
		if err != nil {
			return err
		}
	}
	sensRows := []utils.SensitiveHit{}
	if searchSensitive {
		sensRows, err = querySensitive(db, opt.UrlLike)
		if err != nil {
			return err
		}
	}
	mapRows := []sourceMapRow{}
	if searchMaps {
		mapRows, err = querySourceMaps(db, opt.UrlLike)
		if err != nil {
			return err
		}
	}

	type hit struct {
		Source string
		Field  string
		Value  string
	}

	uniq := make(map[string]hit)
	addHit := func(h hit) {
		key := h.Source + "|" + h.Field + "|" + h.Value
		uniq[key] = h
	}

	for _, r := range apiRows {
		if re.MatchString(r.Path) {
			addHit(hit{Source: r.Root, Field: "api.path", Value: r.Path})
		}
	}
	for _, r := range sensRows {
		if re.MatchString(r.Content) {
			addHit(hit{Source: r.SourceURL, Field: "sensitive.content", Value: r.Content})
		}
	}
	for _, r := range mapRows {
		if re.MatchString(r.MapURL) {
			addHit(hit{Source: r.Root, Field: "source_map.url", Value: r.MapURL})
		}
		if re.MatchString(r.JSURL) {
			addHit(hit{Source: r.Root, Field: "source_map.js", Value: r.JSURL})
		}
	}

	var hits []hit
	for _, h := range uniq {
		hits = append(hits, h)
	}
	sort.Slice(hits, func(i, j int) bool { return hits[i].Source < hits[j].Source })

	if len(hits) == 0 {
		utils.Info("No matches (pattern=%q, db=%s)", opt.Pattern, opt.DbPath)
		return nil
	}

	limit := opt.MaxResult
	if limit <= 0 || limit > len(hits) {
		limit = len(hits)
	}
	hits = hits[:limit]

	for _, h := range hits {
		fmt.Printf("%s\t[%s]\t%s\n", h.Source, h.Field, h.Value)
	}

	// write json
	type out struct {
		Source string `json:"source"`
		Field  string `json:"field"`
		Value  string `json:"value"`
	}
	var outList []out
	for _, h := range hits {
		outList = append(outList, out{Source: h.Source, Field: h.Field, Value: h.Value})
	}
	data, _ := json.MarshalIndent(outList, "", "  ")
	outPath := filepath.Join(viper.GetString("output-dir"), "search_results.json")
	if err := os.WriteFile(outPath, data, 0644); err == nil {
		utils.Info("search results saved: %s", outPath)
	}
	return nil
}

type apiRow struct {
	Root string
	Path string
}

type sourceMapRow struct {
	Root   string
	JSURL  string
	MapURL string
}

func queryAPIPaths(db *sql.DB, like string) ([]apiRow, error) {
	var (
		rows *sql.Rows
		err  error
	)
	if like != "" {
		rows, err = db.Query(`SELECT root_url, path FROM api_paths WHERE root_url LIKE ?`, "%"+like+"%")
	} else {
		rows, err = db.Query(`SELECT root_url, path FROM api_paths`)
	}
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []apiRow
	for rows.Next() {
		var r apiRow
		if err := rows.Scan(&r.Root, &r.Path); err != nil {
			return nil, err
		}
		out = append(out, r)
	}
	return out, rows.Err()
}

func querySensitive(db *sql.DB, like string) ([]utils.SensitiveHit, error) {
	var (
		rows *sql.Rows
		err  error
	)
	if like != "" {
		rows, err = db.Query(`SELECT source_url, category, content, entropy, save_dir FROM sensitive_hits WHERE source_url LIKE ?`, "%"+like+"%")
	} else {
		rows, err = db.Query(`SELECT source_url, category, content, entropy, save_dir FROM sensitive_hits`)
	}
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []utils.SensitiveHit
	for rows.Next() {
		var s utils.SensitiveHit
		if err := rows.Scan(&s.SourceURL, &s.Category, &s.Content, &s.Entropy, &s.SaveDir); err != nil {
			return nil, err
		}
		out = append(out, s)
	}
	return out, rows.Err()
}

func querySourceMaps(db *sql.DB, like string) ([]sourceMapRow, error) {
	var (
		rows *sql.Rows
		err  error
	)
	if like != "" {
		rows, err = db.Query(`SELECT root_url, js_url, map_url FROM source_maps WHERE root_url LIKE ? OR map_url LIKE ?`, "%"+like+"%", "%"+like+"%")
	} else {
		rows, err = db.Query(`SELECT root_url, js_url, map_url FROM source_maps`)
	}
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []sourceMapRow
	for rows.Next() {
		var r sourceMapRow
		if err := rows.Scan(&r.Root, &r.JSURL, &r.MapURL); err != nil {
			return nil, err
		}
		out = append(out, r)
	}
	return out, rows.Err()
}
