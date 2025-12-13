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
	Columns    []int
	ColumnStr  string
}

var searchOptions SearchOptions

func init() {
	searchCmd := &cobra.Command{
		Use:     "grep [pattern]",
		Aliases: []string{"search"},
		Short:   "Regex search in spider.db (api/sensitive/map/page)",
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
	searchCmd.Flags().StringVar(&searchOptions.ColumnStr, "columns", "1,2,3", "select output columns (1=source,2=field,3=value), e.g. --columns 1,3")
	rootCmd.AddCommand(searchCmd)
}

func runSearch(opt SearchOptions) error {
	if err := validateSearchOpts(opt); err != nil {
		return err
	}
	cols, err := parseColumns(opt.ColumnStr)
	if err != nil {
		return err
	}
	opt.Columns = cols
	db, err := utils.InitSpiderDB(opt.DbPath)
	if err != nil {
		return fmt.Errorf("open db failed: %w", err)
	}
	defer db.Close()

	re, err := buildRegex(opt.Pattern, opt.IgnoreCase)
	if err != nil {
		return err
	}

	cfg := searchTables{api: true, sensitive: true, maps: true, pages: true}
	apiRows, sensRows, mapRows, pageRows, err := loadSearchRows(db, opt.UrlLike, cfg)
	if err != nil {
		return err
	}

	hits := collectHits(re, apiRows, sensRows, mapRows, pageRows)
	if len(hits) == 0 {
		utils.Info("No matches (pattern=%q, db=%s)", opt.Pattern, opt.DbPath)
		return nil
	}
	hits = limitHits(hits, opt.MaxResult)
	printHits(hits, opt.Columns)
	writeHitsJSON(hits)
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

type pageRow struct {
	Root    string
	URL     string
	Headers string
	Body    string
}

type hit struct {
	Source string
	Field  string
	Value  string
}

type searchTables struct {
	api       bool
	sensitive bool
	maps      bool
	pages     bool
}

func validateSearchOpts(opt SearchOptions) error {
	if opt.Pattern == "" {
		return fmt.Errorf("pattern is required")
	}
	if _, err := os.Stat(opt.DbPath); err != nil {
		return fmt.Errorf("db not found: %s", opt.DbPath)
	}
	return nil
}

func buildRegex(pattern string, ignoreCase bool) (*regexp.Regexp, error) {
	flags := ""
	if ignoreCase {
		flags = "(?i)"
	}
	return regexp.Compile(flags + pattern)
}

func parseColumns(colStr string) ([]int, error) {
	if strings.TrimSpace(colStr) == "" {
		return []int{1, 2, 3}, nil
	}
	parts := strings.Split(colStr, ",")
	seen := make(map[int]struct{})
	var cols []int
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p == "" {
			continue
		}
		switch p {
		case "1", "source":
			seen[1] = struct{}{}
		case "2", "field":
			seen[2] = struct{}{}
		case "3", "value":
			seen[3] = struct{}{}
		default:
			return nil, fmt.Errorf("invalid column %q (use 1,2,3 or names source,field,value)", p)
		}
	}
	for i := 1; i <= 3; i++ {
		if _, ok := seen[i]; ok {
			cols = append(cols, i)
		}
	}
	if len(cols) == 0 {
		cols = []int{1, 2, 3}
	}
	return cols, nil
}

func loadSearchRows(db *sql.DB, like string, cfg searchTables) ([]apiRow, []utils.SensitiveHit, []sourceMapRow, []pageRow, error) {
	var (
		apiRows  []apiRow
		sensRows []utils.SensitiveHit
		mapRows  []sourceMapRow
		pageRows []pageRow
		err      error
	)
	if cfg.api {
		apiRows, err = queryAPIPaths(db, like)
		if err != nil {
			return nil, nil, nil, nil, err
		}
	}
	if cfg.sensitive {
		sensRows, err = querySensitive(db, like)
		if err != nil {
			return nil, nil, nil, nil, err
		}
	}
	if cfg.maps {
		mapRows, err = querySourceMaps(db, like)
		if err != nil {
			return nil, nil, nil, nil, err
		}
	}
	if cfg.pages {
		pageRows, err = queryPages(db, like)
		if err != nil {
			return nil, nil, nil, nil, err
		}
	}
	return apiRows, sensRows, mapRows, pageRows, nil
}

func collectHits(re *regexp.Regexp, apiRows []apiRow, sensRows []utils.SensitiveHit, mapRows []sourceMapRow, pageRows []pageRow) []hit {
	uniq := make(map[string]hit)
	addHit := func(h hit) {
		uniq[h.Source+"|"+h.Field+"|"+h.Value] = h
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
	for _, r := range pageRows {
		if re.MatchString(r.Body) {
			addHit(hit{Source: r.Root, Field: "page.body", Value: r.Body})
		}
		if re.MatchString(r.Headers) {
			addHit(hit{Source: r.Root, Field: "page.header", Value: r.Headers})
		}
	}

	hits := make([]hit, 0, len(uniq))
	for _, h := range uniq {
		hits = append(hits, h)
	}
	sort.Slice(hits, func(i, j int) bool { return hits[i].Source < hits[j].Source })
	return hits
}

func limitHits(h []hit, max int) []hit {
	if max <= 0 || max > len(h) {
		return h
	}
	return h[:max]
}

func printHits(hits []hit, cols []int) {
	for _, h := range hits {
		parts := make([]string, 0, len(cols))
		for _, c := range cols {
			switch c {
			case 1:
				parts = append(parts, h.Source)
			case 2:
				parts = append(parts, "["+h.Field+"]")
			case 3:
				parts = append(parts, h.Value)
			}
		}
		fmt.Println(strings.Join(parts, "\t"))
	}
}

func writeHitsJSON(hits []hit) {
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

func queryPages(db *sql.DB, like string) ([]pageRow, error) {
	var (
		rows *sql.Rows
		err  error
	)
	if like != "" {
		rows, err = db.Query(`SELECT root_url, url, headers, body FROM page_snapshots WHERE root_url LIKE ? OR url LIKE ?`, "%"+like+"%", "%"+like+"%")
	} else {
		rows, err = db.Query(`SELECT root_url, url, headers, body FROM page_snapshots`)
	}
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []pageRow
	for rows.Next() {
		var r pageRow
		if err := rows.Scan(&r.Root, &r.URL, &r.Headers, &r.Body); err != nil {
			return nil, err
		}
		out = append(out, r)
	}
	return out, rows.Err()
}
