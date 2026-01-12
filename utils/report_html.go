package utils

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"html/template"
	"net/url"
	"os"
	"strings"
	"time"

	"path/filepath"

	"github.com/godspeedcurry/godscan/common"
)

type HTMLReportData struct {
	GeneratedAt   string
	Summary       []SpiderRecord
	APIs          []APIPathRow
	Sensitive     []SensitiveHit
	SourceMaps    []SourceMapHit
	Pages         []PageSnapshotMeta
	PageBodies    []PageSnapshotLite
	Scores        []ScoredRow
	CDNHosts      []CDNHostRow
	Graph         []GraphEdge
	ImportantAPIs []string
	LLMSummary    *LLMSummary
}

type ScoredRow struct {
	RootURL   string         `json:"root_url"`
	Score     int            `json:"score"`
	Status    int            `json:"status"`
	ApiCount  int            `json:"api_count"`
	Important int            `json:"important_api"`
	UrlCount  int            `json:"url_count"`
	CDNCount  int            `json:"cdn_count"`
	SaveDir   string         `json:"save_dir"`
	Reasons   []string       `json:"reasons"`
	RiskFlags []string       `json:"risk_flags"`
	DebugMeta map[string]any `json:"debug_meta"`
}

func ExportHTMLReport(ctx context.Context, db *sql.DB, outputPath string, llmCfg *LLMConfig) error {
	summary, err := LoadSpiderSummaries(db)
	if err != nil {
		return err
	}
	apis, err := LoadAPIPaths(db)
	if err != nil {
		return err
	}
	sens, err := LoadSensitiveHits(db)
	if err != nil {
		return err
	}
	smaps, err := LoadSourceMaps(db)
	if err != nil {
		return err
	}
	pages, err := LoadPageSnapshotMeta(db)
	if err != nil {
		return err
	}
	pageBodies, err := LoadPageSnapshotLite(db)
	if err != nil {
		return err
	}
	cdns, err := LoadCDNHosts(db)
	if err != nil {
		return err
	}

	graph, _ := LoadGraphDB(db, 2000)
	if len(graph) == 0 {
		graph, _ = LoadGraphJSON(defaultGraphPath(outputPath))
	}

	data := HTMLReportData{
		GeneratedAt:   time.Now().Format(time.RFC3339),
		Summary:       summary,
		APIs:          apis,
		Sensitive:     sens,
		SourceMaps:    smaps,
		Pages:         pages,
		PageBodies:    pageBodies,
		CDNHosts:      cdns,
		Graph:         graph,
		ImportantAPIs: common.ImportantApi,
	}
	data.Scores = buildScores(data)
	if ctx == nil {
		ctx = context.Background()
	}
	data.LLMSummary = SummarizeReport(ctx, data, llmCfg)
	return renderHTMLReport(outputPath, data)
}

func defaultGraphPath(htmlPath string) string {
	dir := "."
	if htmlPath != "" {
		dir = filepath.Dir(htmlPath)
	}
	return filepath.Join(dir, "graph.json")
}

func isImportantAPI(path string, patterns []string) bool {
	p := strings.ToLower(path)
	for _, pat := range patterns {
		if pat == "" {
			continue
		}
		if strings.Contains(p, strings.ToLower(pat)) {
			return true
		}
	}
	return false
}

func buildScores(data HTMLReportData) []ScoredRow {
	mapCounts := make(map[string]int)
	for _, m := range data.SourceMaps {
		mapCounts[m.RootURL]++
	}
	importantCounts := make(map[string]int)
	for _, api := range data.APIs {
		if isImportantAPI(api.Path, data.ImportantAPIs) {
			root := api.RootURL
			if root == "" {
				root = rootOf(api.SourceURL)
			}
			importantCounts[root]++
		}
	}
	sensCounts := make(map[string]int)
	for _, s := range data.Sensitive {
		root := rootOf(s.SourceURL)
		if root != "" {
			sensCounts[root]++
		}
	}
	pageLengths := make(map[string]int)
	for _, p := range data.Pages {
		if p.Length > pageLengths[p.RootURL] {
			pageLengths[p.RootURL] = p.Length
		}
		if p.RootURL == "" {
			r := rootOf(p.URL)
			if p.Length > pageLengths[r] {
				pageLengths[r] = p.Length
			}
		}
	}

	var out []ScoredRow
	for _, rec := range data.Summary {
		score := 50
		var reasons []string
		var risks []string
		impCount := importantCounts[rec.Url]
		debug := map[string]any{
			"status":          rec.Status,
			"api_count":       rec.ApiCount,
			"url_count":       rec.UrlCount,
			"cdn_count":       rec.CDNCount,
			"icon_hash":       rec.IconHash,
			"finger":          "", // filled if available in summary later
			"map_count":       mapCounts[rec.Url],
			"sensitive_count": sensCounts[rec.Url],
			"page_length":     pageLengths[rec.Url],
			"important_api":   impCount,
		}

		switch {
		case rec.Status >= 200 && rec.Status < 300:
			score += 10
			reasons = append(reasons, "2xx response")
		case rec.Status >= 300 && rec.Status < 400:
			score -= 15
			risks = append(risks, "redirect/3xx")
		case rec.Status == 401 || rec.Status == 403:
			score -= 25
			risks = append(risks, "auth required")
		case rec.Status == 404:
			score -= 30
			risks = append(risks, "not found")
		case rec.Status >= 500:
			score -= 20
			risks = append(risks, "5xx")
		}

		if rec.ApiCount > 0 {
			score += 15
			reasons = append(reasons, "has APIs")
		}
		if maps := mapCounts[rec.Url]; maps > 0 {
			score += 20
			reasons = append(reasons, "has sourcemap")
		}
		if imp := impCount; imp > 0 {
			score += 15
			reasons = append(reasons, "has important API")
		}
		if sens := sensCounts[rec.Url]; sens > 0 {
			score += 10
			reasons = append(reasons, "has sensitive hits")
			risks = append(risks, "sensitive data")
		}
		if l := pageLengths[rec.Url]; l == 0 {
			score -= 20
			risks = append(risks, "thin content")
		}

		if score < 0 {
			score = 0
		}
		if score > 100 {
			score = 100
		}
		out = append(out, ScoredRow{
			RootURL:   rec.Url,
			Score:     score,
			Status:    rec.Status,
			ApiCount:  rec.ApiCount,
			Important: impCount,
			UrlCount:  rec.UrlCount,
			CDNCount:  rec.CDNCount,
			SaveDir:   rec.SaveDir,
			Reasons:   reasons,
			RiskFlags: risks,
			DebugMeta: debug,
		})
	}
	return out
}

func rootOf(raw string) string {
	u, err := url.Parse(raw)
	if err != nil || u.Scheme == "" || u.Host == "" {
		return ""
	}
	return u.Scheme + "://" + u.Host
}

func renderHTMLReport(path string, data HTMLReportData) error {
	jsonBytes, err := json.Marshal(data)
	if err != nil {
		return err
	}
	tpl, err := template.New("report").Funcs(template.FuncMap{
		"toJSON": func(v interface{}) (template.JS, error) {
			b, e := json.Marshal(v)
			return template.JS(b), e
		},
	}).Parse(reportHTMLTemplate)
	if err != nil {
		return err
	}
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer f.Close()
	payload := struct {
		DataJSON template.JS
	}{
		DataJSON: template.JS(jsonBytes),
	}
	if err := tpl.Execute(f, payload); err != nil {
		return fmt.Errorf("render html: %w", err)
	}
	return nil
}

const reportHTMLTemplate = `<!DOCTYPE html>
<html lang="en">
<head>
  <meta charset="UTF-8">
  <meta name="viewport" content="width=device-width, initial-scale=1">
  <title>godscan report</title>
  <style>
    :root {
      --bg: #0b1021;
      --panel: #11152b;
      --text: #e3e8f7;
      --muted: #8aa0c2;
      --accent: #5de4c7;
      --accent-2: #add7ff;
      --danger: #ff6b81;
    }
    body { margin: 0; font-family: "Inter", "JetBrains Mono", ui-monospace, SFMono-Regular, Menlo, Consolas, monospace; background: radial-gradient(circle at 20% 20%, rgba(93,228,199,0.08), transparent 30%), radial-gradient(circle at 80% 0%, rgba(173,215,255,0.08), transparent 35%), var(--bg); color: var(--text); }
    header { padding: 16px 24px; background: rgba(17,21,43,0.9); position: sticky; top: 0; backdrop-filter: blur(8px); z-index: 10; display: flex; align-items: center; justify-content: space-between; }
    h1 { margin: 0; font-size: 18px; letter-spacing: 0.02em; }
    .pill { display: inline-flex; align-items: center; gap: 6px; padding: 6px 10px; border-radius: 12px; background: rgba(93,228,199,0.12); color: var(--accent); font-size: 12px; }
    main { padding: 16px 24px 60px; }
    .tabs { display: flex; gap: 10px; padding: 12px 24px 0; position: sticky; top: 64px; background: linear-gradient(180deg, rgba(11,16,33,0.95) 0%, rgba(11,16,33,0.6) 60%, transparent); z-index: 9; }
    .tabs button { padding: 8px 12px; border-radius: 10px; border: 1px solid rgba(255,255,255,0.08); background: rgba(255,255,255,0.04); color: var(--text); cursor: pointer; }
    .tabs button.active { background: rgba(93,228,199,0.15); color: var(--accent); border-color: rgba(93,228,199,0.4); }
    section { margin-bottom: 28px; }
    .section { display: none; }
    .section.active { display: block; }
    .panel { background: var(--panel); border: 1px solid rgba(255,255,255,0.05); border-radius: 12px; padding: 12px; box-shadow: 0 10px 40px rgba(0,0,0,0.25); }
    .panel header { position: relative; background: transparent; padding: 0; margin-bottom: 8px; }
    .panel h2 { margin: 0; font-size: 16px; color: var(--accent-2); }
    .controls { display: flex; align-items: center; gap: 10px; flex-wrap: wrap; margin-top: 6px; }
    .controls input[type="search"] { padding: 8px 10px; border-radius: 8px; border: 1px solid rgba(255,255,255,0.08); background: rgba(255,255,255,0.04); color: var(--text); min-width: 200px; }
    .controls select, .controls button { padding: 8px 10px; border-radius: 8px; border: 1px solid rgba(255,255,255,0.08); background: rgba(255,255,255,0.06); color: var(--text); }
    .controls input[type="text"] { padding: 8px 10px; border-radius: 8px; border: 1px solid rgba(93,228,199,0.25); background: rgba(93,228,199,0.1); color: var(--text); min-width: 200px; }
    table { width: 100%; border-collapse: collapse; }
    th, td { padding: 8px 10px; border-bottom: 1px solid rgba(255,255,255,0.05); font-size: 12px; }
    .ellipsis { max-width: 520px; display: inline-block; white-space: nowrap; overflow: hidden; text-overflow: ellipsis; vertical-align: bottom; }
    .ellipsis-long { max-width: 420px; display: inline-block; white-space: nowrap; overflow: hidden; text-overflow: ellipsis; vertical-align: bottom; }
    .icon-thumb { width: 20px; height: 20px; object-fit: contain; border: 1px solid #1f2540; border-radius: 4px; padding: 2px; background: #0b1021; box-shadow: 0 0 0 1px rgba(93,228,199,0.08); }
    .icon-cell { display:flex; align-items:center; gap:8px; }
    .mini-copy { padding: 2px 6px; border-radius: 6px; border: 1px solid rgba(255,255,255,0.12); background: rgba(255,255,255,0.06); color: var(--text); font-size: 10px; cursor: pointer; }
    .mini-copy:hover { border-color: var(--accent); color: var(--accent); }
    .hash-cell { position: relative; }
    .hash-bubble { position: absolute; top: 100%; left: 0; margin-top: 4px; background: rgba(15,19,38,0.95); border: 1px solid rgba(93,228,199,0.25); box-shadow: 0 8px 30px rgba(0,0,0,0.4); padding: 8px; border-radius: 8px; display: none; z-index: 15; min-width: 280px; max-width: 560px; }
    .hash-cell:hover .hash-bubble { display: block; }
    .hash-bubble pre { margin: 0 0 6px 0; white-space: pre-wrap; font-family: inherit; font-size: 12px; color: var(--text); }
    .copy-btn { padding: 4px 8px; border-radius: 6px; border: 1px solid rgba(255,255,255,0.08); background: rgba(255,255,255,0.05); color: var(--text); cursor: pointer; font-size: 11px; }
    .copy-icon { margin-left: 8px; cursor: pointer; color: var(--muted); font-size: 13px; user-select: none; }
    .copy-icon:hover { color: var(--accent); }
    .toast { position: fixed; bottom: 16px; right: 16px; background: rgba(17,21,43,0.9); color: var(--text); padding: 10px 12px; border-radius: 10px; border: 1px solid rgba(255,255,255,0.08); font-size: 12px; box-shadow: 0 10px 30px rgba(0,0,0,0.25); display: none; z-index: 30; }
    th { text-align: left; color: var(--muted); font-weight: 600; cursor: pointer; }
    th[data-sort="asc"]::after { content: " ↑"; color: var(--accent); }
    th[data-sort="desc"]::after { content: " ↓"; color: var(--accent); }
    tbody tr:hover { background: rgba(255,255,255,0.03); }
    .status { color: var(--accent); }
    .small { color: var(--muted); font-size: 12px; }
    .stats { display: grid; grid-template-columns: repeat(auto-fit,minmax(180px,1fr)); gap: 10px; margin-bottom: 12px; }
    .stat { background: rgba(255,255,255,0.04); border: 1px solid rgba(255,255,255,0.06); padding: 10px; border-radius: 10px; }
    .stat .label { color: var(--muted); font-size: 12px; }
    .stat .value { font-size: 18px; font-weight: 700; }
    .pagination { display: flex; gap: 8px; align-items: center; color: var(--muted); font-size: 12px; }
    .pagination button { padding: 6px 10px; border-radius: 6px; border: 1px solid rgba(255,255,255,0.08); background: rgba(255,255,255,0.05); color: var(--text); cursor: pointer; }
    .tag { display: inline-block; padding: 2px 6px; border-radius: 6px; background: rgba(255,255,255,0.06); font-size: 11px; color: var(--muted); }
    .btn { padding: 6px 10px; border-radius: 8px; border: 1px solid rgba(93,228,199,0.25); background: rgba(93,228,199,0.12); color: var(--accent); cursor: pointer; font-size: 12px; }
    .tag-box { display: flex; gap: 6px; flex-wrap: wrap; align-items: center; }
    .tag-item { background: rgba(93,228,199,0.16); color: var(--accent); border: 1px solid rgba(93,228,199,0.35); padding: 3px 6px; border-radius: 8px; display: inline-flex; align-items: center; gap: 6px; font-size: 11px; box-shadow: 0 4px 18px rgba(0,0,0,0.2); }
    .tag-remove { cursor: pointer; color: var(--muted); }
    .tag-remove:hover { color: var(--accent); }
    .drawer-overlay { position: fixed; inset: 0; background: rgba(0,0,0,0.55); backdrop-filter: blur(4px); display: none; align-items: center; justify-content: center; z-index: 20; }
    .drawer { width: min(960px, 95vw); max-height: 90vh; overflow: auto; background: var(--panel); border: 1px solid rgba(255,255,255,0.08); border-radius: 14px; padding: 16px; box-shadow: 0 12px 60px rgba(0,0,0,0.4); }
    .drawer h3 { margin: 0 0 8px 0; color: var(--accent-2); }
    .drawer .row { margin-bottom: 8px; color: var(--muted); font-size: 13px; }
    .drawer code { background: rgba(255,255,255,0.05); padding: 2px 4px; border-radius: 4px; }
    .list-block { margin: 8px 0; padding: 8px; border: 1px solid rgba(255,255,255,0.05); border-radius: 8px; background: rgba(255,255,255,0.03); }
    .list-block h4 { margin: 0 0 6px 0; font-size: 13px; color: var(--accent); }
    .tooltip { position: fixed; background: rgba(10,14,28,0.95); color: var(--text); padding: 8px 10px; border-radius: 8px; border: 1px solid rgba(93,228,199,0.25); box-shadow: 0 10px 30px rgba(0,0,0,0.35); pointer-events: none; font-size: 12px; display: none; max-width: 520px; z-index: 40; }
    .llm-hero { background: linear-gradient(120deg, rgba(93,228,199,0.35) 0%, rgba(173,215,255,0.3) 45%, rgba(93,228,199,0.28) 100%); border: 1px solid rgba(173,215,255,0.6); padding: 16px; border-radius: 14px; display:flex; align-items:center; justify-content:space-between; gap:12px; box-shadow: 0 18px 60px rgba(0,0,0,0.45); color: var(--bg); }
    .llm-title { font-size: 20px; font-weight: 700; margin: 0 0 6px 0; color: var(--bg); text-shadow: 0 1px 0 rgba(255,255,255,0.3); }
    .llm-kicker { text-transform: uppercase; letter-spacing: 0.08em; font-size: 11px; color: #0b1021; opacity: 0.8; }
    .llm-meta { color: var(--bg); opacity: 0.9; font-size: 13px; }
    .llm-grid { display: grid; grid-template-columns: 1.2fr 0.8fr; gap: 12px; margin-top: 12px; }
    .llm-card { background: rgba(255,255,255,0.02); border: 1px solid rgba(255,255,255,0.08); border-radius: 12px; padding: 12px; box-shadow: 0 10px 40px rgba(0,0,0,0.25); }
    .llm-card-head { font-size: 14px; color: var(--accent-2); margin-bottom: 6px; font-weight: 600; }
    .llm-output { white-space: pre-wrap; line-height: 1.6; font-size: 13px; color: var(--text); }
    .llm-pre { white-space: pre-wrap; background: rgba(255,255,255,0.04); padding: 10px; border-radius: 8px; border: 1px solid rgba(255,255,255,0.05); color: var(--muted); font-size: 12px; }
    .llm-subhead { color: var(--muted); font-size: 12px; margin: 6px 0 4px; }
    .llm-error { margin-top: 8px; background: rgba(255,107,129,0.12); border: 1px solid rgba(255,107,129,0.4); color: var(--text); padding: 10px; border-radius: 10px; font-size: 12px; }
    .md-table { width:100%; border-collapse:collapse; margin-top:6px; }
    .md-table th, .md-table td { padding:6px 8px; border:1px solid rgba(255,255,255,0.12); font-size:12px; }
    .md-table th { background: rgba(255,255,255,0.05); color: var(--accent-2); text-align:left; }
    @media(max-width: 900px) { .llm-grid { grid-template-columns: 1fr; } .llm-hero { flex-direction: column; align-items: flex-start; } }
  </style>
</head>
<body>
  <header>
    <h1>godscan HTML report</h1>
    <div class="pill">Generated at <span id="generated-at"></span></div>
  </header>
  <nav class="tabs">
    <button data-target="section-llm">LLM Output</button>
    <button class="active" data-target="section-score">Scores</button>
    <button data-target="section-summary">Summary</button>
    <button data-target="section-important">Important APIs</button>
    <button data-target="section-api">APIs</button>
    <button data-target="section-sensitive">Sensitive</button>
    <button data-target="section-maps">SourceMaps</button>
    <button data-target="section-pages">Pages</button>
    <button data-target="section-graph">Graph</button>
  </nav>
  <main>
    <section class="panel section" id="section-llm">
      <div class="llm-hero">
        <div>
          <div class="llm-kicker">LLM Briefing</div>
          <div class="llm-title" data-llm-status>模型摘要未启用</div>
          <div class="llm-meta" data-llm-meta>提示：生成报表时提供 --llm-key 或环境变量 GODSCAN_LLM_KEY</div>
        </div>
        <div class="pill" data-llm-stamp>离线报表</div>
      </div>
      <div class="llm-grid">
        <div class="llm-card">
          <div class="llm-card-head">简洁问题摘要</div>
          <div class="llm-output" data-llm-output>暂无摘要，生成报表时携带 LLM key 后自动生成。</div>
        </div>
        <div class="llm-card">
          <div class="llm-card-head">Prompt & 输入片段</div>
          <div class="llm-subhead">Prompt</div>
          <pre class="llm-pre" data-llm-prompt></pre>
          <div class="llm-subhead">输入（截断预览）</div>
          <pre class="llm-pre" data-llm-input>生成报表时会截取爬虫文本拼接到此处。</pre>
        </div>
      </div>
      <div class="llm-error" data-llm-error style="display:none;"></div>
    </section>

    <section class="panel section active" id="section-score">
      <header>
        <h2>Scores (rule-based)</h2>
        <div class="controls">
          <input id="score-search" type="search" placeholder="Filter root/reason/risk">
          <select id="score-page-size">
            <option value="200">200 / page</option>
            <option value="500">500 / page</option>
            <option value="1000">1000 / page</option>
          </select>
          <div class="pagination" id="score-pagination"></div>
        </div>
      </header>
      <div style="overflow:auto">
        <table data-table="scores">
          <thead id="score-head"><tr><th data-col="root_url">Root</th><th data-col="score">Score</th><th data-col="status">Status</th><th data-col="api_count">API</th><th data-col="important_api">Important API</th><th data-col="url_count">URLs</th><th data-col="cdn_count">CDN</th><th data-col="reasons">Reasons</th><th data-col="risk_flags">Risk</th><th data-col="save_dir">Save Dir</th><th>Detail</th></tr></thead>
          <tbody id="score-body"></tbody>
        </table>
      </div>
    </section>

    <section class="panel section" id="section-summary">
      <header>
        <h2>Summary</h2>
        <div class="controls">
          <input id="summary-search" type="search" placeholder="Filter by URL/icon/status">
          <div class="pagination" id="summary-pagination"></div>
        </div>
      </header>
      <div class="stats" id="summary-stats"></div>
      <div style="overflow:auto">
        <table data-table="summary">
          <thead id="summary-head">
            <tr>
              <th data-col="Url">URL</th><th data-col="IconBase64">Icon</th><th data-col="IconHash">Icon hash</th><th data-col="ApiCount">API</th><th data-col="UrlCount">URLs</th><th data-col="CDNCount">CDN URLs</th><th data-col="CDNHosts">CDN Hosts</th><th data-col="Status">Status</th><th data-col="SaveDir">Save Dir</th>
            </tr>
          </thead>
          <tbody id="summary-body"></tbody>
        </table>
      </div>
    </section>

    {{/* APIs */}}
    <section class="panel section" id="section-important">
      <header>
        <h2>Important APIs</h2>
        <div class="controls">
          <input id="important-search" type="search" placeholder="Filter root/path/source">
          <div class="tag-box" id="important-exclude-tags"></div>
          <input id="important-exclude-input" type="text" placeholder="Exclude keywords (Enter to add)">
          <select id="important-page-size">
            <option value="200">200 / page</option>
            <option value="500">500 / page</option>
            <option value="1000">1000 / page</option>
          </select>
          <div class="pagination" id="important-pagination"></div>
        </div>
      </header>
      <div style="overflow:auto">
        <table data-table="important">
          <thead id="important-head"><tr><th data-col="root_url">Root</th><th data-col="path">Path</th><th data-col="source_url">Source</th><th data-col="save_dir">Save Dir</th></tr></thead>
          <tbody id="important-body"></tbody>
        </table>
      </div>
    </section>

    {{/* APIs */}}
    <section class="panel section" id="section-api">
      <header>
        <h2>APIs</h2>
        <div class="controls">
          <input id="api-search" type="search" placeholder="Filter root/path/source">
          <div class="tag-box" id="api-exclude-tags"></div>
          <input id="api-exclude-input" type="text" placeholder="Exclude keywords (Enter to add)">
          <select id="api-page-size">
            <option value="200">200 / page</option>
            <option value="500">500 / page</option>
            <option value="1000">1000 / page</option>
          </select>
          <div class="pagination" id="api-pagination"></div>
        </div>
      </header>
      <div style="overflow:auto">
        <table data-table="api">
          <thead id="api-head"><tr><th data-col="root_url">Root</th><th data-col="path">Path</th><th data-col="source_url">Source</th><th data-col="save_dir">Save Dir</th></tr></thead>
          <tbody id="api-body"></tbody>
        </table>
      </div>
    </section>

    {{/* Sensitive */}}
    <section class="panel section" id="section-sensitive">
      <header>
        <h2>Sensitive</h2>
      </header>
      <div class="panel" style="margin-bottom:12px;">
        <header>
          <h3 style="margin:0;font-size:14px;color:var(--accent-2);">security-rule-* (high risk)</h3>
          <div class="controls">
            <input id="sens-rule-search" type="search" placeholder="Filter category/content/source">
            <select id="sens-rule-page-size">
              <option value="200">200 / page</option>
              <option value="500">500 / page</option>
              <option value="1000">1000 / page</option>
            </select>
            <div class="pagination" id="sens-rule-pagination"></div>
          </div>
        </header>
        <div style="overflow:auto">
          <table data-table="sensitive_rules">
            <thead id="sens-rule-head"><tr><th data-col="category">Category</th><th data-col="content">Content</th><th data-col="source_url">Source</th><th data-col="entropy">Entropy</th><th data-col="save_dir">Save Dir</th></tr></thead>
            <tbody id="sens-rule-body"></tbody>
          </table>
        </div>
      </div>
      <div class="panel">
        <header>
          <h3 style="margin:0;font-size:14px;color:var(--accent-2);">Other sensitive</h3>
          <div class="controls">
            <input id="sens-other-search" type="search" placeholder="Filter category/content/source">
            <select id="sens-other-page-size">
              <option value="200">200 / page</option>
              <option value="500">500 / page</option>
              <option value="1000">1000 / page</option>
            </select>
            <div class="pagination" id="sens-other-pagination"></div>
          </div>
        </header>
        <div style="overflow:auto">
          <table data-table="sensitive_other">
            <thead id="sens-other-head"><tr><th data-col="category">Category</th><th data-col="content">Content</th><th data-col="source_url">Source</th><th data-col="entropy">Entropy</th><th data-col="save_dir">Save Dir</th></tr></thead>
          <tbody id="sens-other-body"></tbody>
          </table>
        </div>
      </div>
    </section>

    {{/* SourceMaps */}}
    <section class="panel section" id="section-maps">
      <header>
        <h2>Source maps</h2>
        <div class="controls">
          <input id="map-search" type="search" placeholder="Filter js/map/root">
          <select id="map-page-size">
            <option value="200">200 / page</option>
            <option value="500">500 / page</option>
            <option value="1000">1000 / page</option>
          </select>
          <div class="pagination" id="map-pagination"></div>
        </div>
      </header>
      <div style="overflow:auto">
        <table data-table="maps">
          <thead id="map-head"><tr><th data-col="root_url">Root</th><th data-col="js_url">JS</th><th data-col="map_url">Map</th><th data-col="status">Status</th><th data-col="length">Length</th></tr></thead>
          <tbody id="map-body"></tbody>
        </table>
      </div>
    </section>

    {{/* Pages */}}
    <section class="panel section" id="section-pages">
      <header>
        <h2>Pages (metadata)</h2>
        <div class="controls">
          <input id="page-search" type="search" placeholder="Filter URL/content-type/root">
          <select id="page-page-size">
            <option value="200">200 / page</option>
            <option value="500">500 / page</option>
            <option value="1000">1000 / page</option>
          </select>
          <div class="pagination" id="page-pagination"></div>
        </div>
      </header>
      <div style="overflow:auto">
        <table data-table="pages">
          <thead id="page-head"><tr><th data-col="root_url">Root</th><th data-col="url">URL</th><th data-col="status">Status</th><th data-col="content_type">Type</th><th data-col="length">Length</th><th data-col="save_dir">Save Dir</th></tr></thead>
          <tbody id="page-body"></tbody>
        </table>
      </div>
    </section>

    {{/* Graph */}}
    <section class="panel section" id="section-graph">
      <header>
        <h2>Graph (sampled)</h2>
        <div class="controls">
          <span class="small">最多显示 2000 条边 · 颜色=状态 (绿/橙/红/灰) · 形状=类型 (● 页, ▢ 资源, ◆ API, ✚ Map)</span>
        </div>
      </header>
      <div style="display:grid;grid-template-columns:1fr 260px;gap:12px;align-items:start;">
        <div id="graph-container" style="background:rgba(255,255,255,0.02); border:1px solid rgba(255,255,255,0.05); border-radius:12px; padding:8px; overflow:auto;">
          <svg id="graph-svg" width="100%" height="640"></svg>
        </div>
        <div class="panel" style="background:rgba(255,255,255,0.03);">
          <h3 style="margin-top:0;">Legend</h3>
          <div class="small">颜色：绿=2xx 橙=3xx 红=4xx/5xx 灰=未知</div>
          <div class="small" style="margin-bottom:8px;">形状：● 页面 ▢ 资源 ◆ API ✚ Map</div>
          <div id="graph-stats" class="small"></div>
          <div id="graph-top" class="small" style="margin-top:8px;"></div>
        </div>
      </div>
    </section>
  </main>

    <div class="drawer-overlay" id="detail-overlay">
      <div class="drawer" id="detail-drawer">
        <div style="display:flex;align-items:center;justify-content:space-between;margin-bottom:6px;">
          <h3 id="detail-title">Details</h3>
          <button class="btn" id="detail-close">Close</button>
        </div>
        <div id="detail-body"></div>
      </div>
    </div>
  <div class="toast" id="toast"></div>
  <div class="tooltip" id="graph-tip"></div>

  <script>
    const data = {{.DataJSON}};
    const defaultLLMPrompt = "请分析如下爬虫爬到的长文本内容，用简洁的语言输出潜在的问题，优先输出安全风险、认证/调试入口、敏感数据暴露。使用中文分点总结，避免重复。";
    const importantApis = (data.ImportantAPIs || []).map(s => (s || "").toLowerCase());
    const importantAPIData = (data.APIs || []).filter(a => isImportant(a.path));
    const graphEdges = (data.Graph || []).slice(0, 2000);
    const pageMeta = {};
    (data.Pages || []).forEach(p => {
      if (p.url) {
        pageMeta[p.url] = {status:p.status, type:p.content_type};
      }
    });
    const sensitiveRules = (data.Sensitive||[]).filter(s => (s.category||"").toLowerCase().startsWith("security-rule"));
    const sensitiveOther = (data.Sensitive||[]).filter(s => !(s.category||"").toLowerCase().startsWith("security-rule"));
    const fmt = {
      esc: (s) => String(s || "").replace(/[&<>"']/g, c => ({'&':'&amp;','<':'&lt;','>':'&gt;','"':'&quot;',"'":'&#39;'}[c])),
      num: (n) => isFinite(n) ? n.toLocaleString() : n
    };

    function escapeHTML(s) {
      return String(s || "").replace(/[&<>"']/g, c => ({'&':'&amp;','<':'&lt;','>':'&gt;','"':'&quot;'}[c]));
    }

    function renderInline(md) {
      return escapeHTML(md)
        .replace(/\*\*([^*]+)\*\*/g, "<strong>$1</strong>")
        .replace(/\x60([^\x60]+)\x60/g, "<code>$1</code>");
    }

    function markdownToHTML(md) {
      if (!md) return "<p>暂无摘要，提供 LLM key 后会自动生成，或检查爬虫是否抓到有效内容。</p>";
      const lines = String(md).split(/\r?\n/);
      let html = "";
      let inUl = false, inOl = false;
      const closeLists = () => {
        if (inUl) { html += "</ul>"; inUl = false; }
        if (inOl) { html += "</ol>"; inOl = false; }
      };
      for (let i = 0; i < lines.length; i++) {
        const line = lines[i].trim();
        if (!line) { closeLists(); continue; }

        const heading = line.match(/^(#{1,6})\s+(.*)$/);
        if (heading) {
          closeLists();
          html += "<h"+heading[1].length+">"+renderInline(heading[2])+"</h"+heading[1].length+">";
          continue;
        }

        if (line.startsWith("|")) {
          const parsed = parseTable(lines, i);
          if (parsed) {
            closeLists();
            html += parsed.html;
            i = parsed.nextIdx;
            continue;
          }
        }

        const ul = line.match(/^(\*|-)\s+(.*)$/);
        if (ul) {
          if (inOl) { html += "</ol>"; inOl = false; }
          if (!inUl) { html += "<ul>"; inUl = true; }
          html += "<li>"+renderInline(ul[2])+"</li>";
          continue;
        }
        const ol = line.match(/^\d+\.\s+(.*)$/);
        if (ol) {
          if (inUl) { html += "</ul>"; inUl = false; }
          if (!inOl) { html += "<ol>"; inOl = true; }
          html += "<li>"+renderInline(ol[1])+"</li>";
          continue;
        }
        closeLists();
        html += "<p>"+renderInline(line)+"</p>";
      }
      closeLists();
      return html || "<p>"+renderInline(md)+"</p>";
    }

    function parseTable(lines, startIdx) {
      const rows = [];
      let idx = startIdx;
      while (idx < lines.length && lines[idx].trim().startsWith("|")) {
        rows.push(lines[idx].trim());
        idx++;
      }
      if (rows.length < 2) return null;
      const header = splitRow(rows[0]);
      const sep = splitRow(rows[1]);
      if (!header.length || header.length !== sep.length || !sep.every(c => /^:?-{3,}:?$/.test(c))) {
        return null;
      }
      const bodyRows = rows.slice(2).map(splitRow).filter(r => r.length);
      const headHtml = header.map(c => "<th>"+renderInline(c)+"</th>").join("");
      const bodyHtml = bodyRows.map(r => "<tr>"+r.map(c => "<td>"+renderInline(c)+"</td>").join("")+"</tr>").join("");
      return {html: '<table class="md-table"><thead><tr>'+headHtml+'</tr></thead><tbody>'+bodyHtml+'</tbody></table>', nextIdx: idx-1};
    }

    function splitRow(row) {
      const trimmed = row.replace(/^\|/, "").replace(/\|$/, "");
      return trimmed.split("|").map(c => c.trim());
    }

    function renderLLM(llm) {
      const section = document.getElementById("section-llm");
      if (!section) return;
      const status = section.querySelector("[data-llm-status]");
      const meta = section.querySelector("[data-llm-meta]");
      const stamp = section.querySelector("[data-llm-stamp]");
      const outEl = section.querySelector("[data-llm-output]");
      const inputEl = section.querySelector("[data-llm-input]");
      const promptEl = section.querySelector("[data-llm-prompt]");
      const errEl = section.querySelector("[data-llm-error]");
      const enabled = !!(llm && (llm.enabled || llm.output || llm.error));
      const provider = (llm && llm.provider) ? llm.provider : "google";
      const model = (llm && llm.model) ? llm.model : "gemini-2.5-flash";
      const outText = (llm && llm.output) ? llm.output : "";
      const prompt = (llm && llm.prompt) ? llm.prompt : defaultLLMPrompt;
      const inputPreview = (llm && llm.input_preview) ? llm.input_preview : "生成报表时会截取爬虫文本拼接后发送给模型，预览会显示在此处。";
      const limitIn = llm && llm.input_limit ? llm.input_limit : null;
      const limitOut = llm && llm.output_limit ? llm.output_limit : null;
      if (status) {
        if (outText) status.textContent = "模型摘要";
        else if (llm && llm.error) status.textContent = "LLM 摘要失败";
        else status.textContent = "LLM 摘要未启用";
      }
      if (meta) {
        if (enabled) {
          const ts = llm && llm.generated_at ? ("生成于 " + llm.generated_at) : "离线渲染";
          let limits = "";
          if (limitIn || limitOut) {
            limits = " · 输入上限 " + (limitIn || "?") + " 字符";
            if (limitOut) limits += " · 输出上限 " + limitOut + " tokens";
          }
          meta.textContent = "Provider " + provider + " · Model " + model + " · " + ts + limits;
        } else {
          meta.textContent = "提示：运行 godscan report --llm-key <token> 或设置 GODSCAN_LLM_KEY 可在报表内生成摘要。";
        }
      }
      if (stamp) {
        stamp.textContent = (llm && llm.generated_at) ? llm.generated_at : "离线报表";
      }
      if (outEl) {
        outEl.innerHTML = markdownToHTML(outText);
      }
      if (promptEl) {
        promptEl.textContent = prompt;
      }
      if (inputEl) {
        inputEl.textContent = inputPreview || "无可用内容";
      }
      if (errEl) {
        if (llm && llm.error) {
          errEl.style.display = "block";
          errEl.textContent = llm.error;
        } else {
          errEl.style.display = "none";
        }
      }
    }

    renderLLM(data.LLMSummary || {});

    function rootOf(u) {
      try { const x = new URL(u); return x.protocol + "//" + x.host; } catch { return ""; }
    }

    function isImportant(path) {
      const p = (path || "").toLowerCase();
      return importantApis.some(k => k && p.includes(k));
    }

    function matchesExcludeRow(row, set, keys) {
      const cols = keys || ["path"];
      for (const k of set) {
        const kw = (k || "").toLowerCase();
        if (!kw) continue;
        for (const col of cols) {
          const v = (row[col] || "").toString().toLowerCase();
          if (v.includes(kw)) return true;
        }
      }
      return false;
    }

    function setupExcludeUI(inputId, tagBoxId, onChange) {
      const input = document.getElementById(inputId);
      const box = document.getElementById(tagBoxId);
      const tags = [];
      const set = new Set();
      function renderTags() {
        if (!box) return;
        box.innerHTML = tags.map((t,i)=> '<span class="tag-item">'+fmt.esc(t)+' <span class="tag-remove" data-idx="'+i+'">×</span></span>').join("");
      }
      function addTag(raw) {
        const v = (raw || "").trim().toLowerCase();
        if (!v) return;
        if (!set.has(v)) {
          set.add(v);
          tags.push(v);
          renderTags();
          onChange && onChange(Array.from(set));
        }
        if (input) input.value = "";
      }
      function removeIdx(idx) {
        const t = tags[idx];
        tags.splice(idx,1);
        set.delete(t);
        renderTags();
        onChange && onChange(Array.from(set));
      }
      if (input) {
        input.addEventListener("keydown", (e) => {
          if (e.key === "Enter" || e.key === ",") {
            e.preventDefault();
            addTag(input.value);
          }
        });
        input.addEventListener("blur", () => addTag(input.value));
      }
      if (box) {
        box.addEventListener("click", (e) => {
          const rm = e.target.closest(".tag-remove");
          if (!rm) return;
          const idx = parseInt(rm.dataset.idx, 10);
          if (!isNaN(idx)) removeIdx(idx);
        });
      }
      renderTags();
      return {set, tags, addTag};
    }

    function setupTable({data, columns, tbodyId, searchId, pagerId, pageSizeId, headId, defaultSize=200, initialSortKey=null, initialSortDir="asc", extraFilter=null}) {
      const tbody = document.getElementById(tbodyId);
      const searchInput = document.getElementById(searchId);
      const pager = document.getElementById(pagerId);
      const pageSizeSel = pageSizeId ? document.getElementById(pageSizeId) : null;
      const head = headId ? document.getElementById(headId) : null;
      let page = 1;
      let pageSize = pageSizeSel ? parseInt(pageSizeSel.value,10) : defaultSize;
      let filter = "";
      let sortKey = initialSortKey;
      let sortDir = initialSortDir;
      const colMap = {};
      columns.forEach(c => { colMap[c.key] = c; });

      function filterRows() {
        const f = filter.toLowerCase();
        return data.filter(row => {
          if (f) {
            const match = columns.some(col => (row[col.key] || "").toString().toLowerCase().includes(f));
            if (!match) return false;
          }
          if (extraFilter && !extraFilter(row)) return false;
          return true;
        });
      }

      function sortRows(rows) {
        if (!sortKey) return rows;
        const col = colMap[sortKey] || {};
        const sorted = rows.slice().sort((a,b) => {
          const va = col.render ? col.render(a) : (col.value ? col.value(a) : a[sortKey]);
          const vb = col.render ? col.render(b) : (col.value ? col.value(b) : b[sortKey]);
          const na = parseFloat(va); const nb = parseFloat(vb);
          const aNum = !isNaN(na) && isFinite(na) && String(va).trim() !== "";
          const bNum = !isNaN(nb) && isFinite(nb) && String(vb).trim() !== "";
          let cmp = 0;
          if (aNum && bNum) {
            cmp = na - nb;
          } else {
            cmp = String(va || "").localeCompare(String(vb || ""), undefined, {numeric:true, sensitivity:"base"});
          }
          return sortDir === "asc" ? cmp : -cmp;
        });
        return sorted;
      }

      function updateSortIndicators() {
        if (!head) return;
        head.querySelectorAll("[data-col]").forEach(th => {
          th.dataset.sort = (th.dataset.col === sortKey) ? sortDir : "";
        });
      }

      function render() {
        const rows = sortRows(filterRows());
        const totalPages = Math.max(1, Math.ceil(rows.length / pageSize));
        if (page > totalPages) page = totalPages;
        const start = (page-1)*pageSize;
        const slice = rows.slice(start, start + pageSize);
        tbody.innerHTML = slice.map(r => "<tr>" + columns.map(col => {
          const val = col.render ? col.render(r) : r[col.key];
          return "<td>" + (col.raw ? val : fmt.esc(val)) + "</td>";
        }).join("") + "</tr>").join("");
        pager.innerHTML =
          '<button ' + (page<=1?'disabled':'') + ' data-nav="prev">Prev</button>' +
          '<span>' + rows.length + ' rows | page ' + page + '/' + totalPages + '</span>' +
          '<button ' + (page>=totalPages?'disabled':'') + ' data-nav="next">Next</button>';
        updateSortIndicators();
      }

      pager.addEventListener("click", (e) => {
        if (e.target.dataset.nav === "prev" && page > 1) { page--; render(); }
        if (e.target.dataset.nav === "next") { page++; render(); }
      });
      if (searchInput) {
        searchInput.addEventListener("input", (e) => { filter = e.target.value.trim(); page=1; render(); });
      }
      if (pageSizeSel) {
        pageSizeSel.addEventListener("change", (e) => { pageSize = parseInt(e.target.value,10); page=1; render(); });
      }
      if (head) {
        head.addEventListener("click", (e) => {
          const th = e.target.closest("[data-col]");
          if (!th || !head.contains(th)) return;
          const key = th.dataset.col;
          if (sortKey === key) {
            sortDir = (sortDir === "asc") ? "desc" : "asc";
          } else {
            sortKey = key;
            sortDir = "asc";
          }
          render();
        });
      }
      render();
      return { rerender: render };
    }

    function renderStats(summary) {
      document.getElementById("generated-at").textContent = data.GeneratedAt || "";
      const totalHosts = summary.length;
      const totalAPIs = summary.reduce((acc, r) => acc + (r.ApiCount || 0), 0);
      const totalURLs = summary.reduce((acc, r) => acc + (r.UrlCount || 0), 0);
      const totalCDN = summary.reduce((acc, r) => acc + (r.CDNCount || 0), 0);
      const stats = document.getElementById("summary-stats");
      stats.innerHTML =
        '<div class="stat"><div class="label">Hosts</div><div class="value">'+fmt.num(totalHosts)+'</div></div>' +
        '<div class="stat"><div class="label">APIs</div><div class="value">'+fmt.num(totalAPIs)+'</div></div>' +
        '<div class="stat"><div class="label">URLs</div><div class="value">'+fmt.num(totalURLs)+'</div></div>' +
        '<div class="stat"><div class="label">CDN URLs</div><div class="value">'+fmt.num(totalCDN)+'</div></div>';
    }

    renderStats(data.Summary || []);

    setupTable({
      data: data.Summary || [],
      columns: [
        {key:"Url", render:(r)=> '<span class="ellipsis" title="'+fmt.esc(r.Url||"")+'">'+fmt.esc(r.Url||"")+'</span>', raw:true},
        {key:"IconBase64", render:(r)=> {
          if (!r.IconBase64) return '<span class="small">-</span>';
          return '<div class="icon-cell" title="'+fmt.esc(r.IconHash||"")+'"><img class="icon-thumb" src="data:image/x-icon;base64,'+fmt.esc(r.IconBase64)+'" alt="icon"><button class="mini-copy" data-hash="'+fmt.esc(r.IconHash||"")+'">copy</button></div>';
        }, raw:true},
        {key:"IconHash", render:(r)=> {
          const hash = r.IconHash || "";
          if (!hash) return '<span class="small">-</span>';
          const esc = fmt.esc(hash);
          return '<div class="hash-cell"><span class="ellipsis-long">'+esc+'</span><div class="hash-bubble"><pre>'+esc+'</pre><button class="mini-copy" data-hash="'+esc+'">copy</button></div></div>';
        }, raw:true},
        {key:"ApiCount"},
        {key:"UrlCount"},
        {key:"CDNCount"},
        {key:"CDNHosts"},
        {key:"Status"},
        {key:"SaveDir"},
      ],
      tbodyId:"summary-body",
      searchId:"summary-search",
      pagerId:"summary-pagination",
      headId:"summary-head",
      defaultSize:200
    });

    const findStats = buildFindStats();

    setupTable({
      data: data.Scores || [],
      columns: [
        {key:"root_url", render:(r)=> '<span class="ellipsis" title="'+fmt.esc(r.root_url||"")+'">'+fmt.esc(r.root_url||"")+'</span>', raw:true},
        {key:"score"},
        {key:"status"},
        {key:"api_count"},
        {key:"important_api"},
        {key:"url_count"},
        {key:"cdn_count"},
        {key:"findings", render:(r)=> {
          const s = findStats[r.root_url] || {};
          return "API:"+ (s.api||0)+" / Important:"+ (s.important||0)+" / Sensitive:"+ (s.sens||0)+" / Maps:"+ (s.maps||0)+" / Pages:"+ (s.pages||0)+" / CDN:"+ (s.cdn||0);
        }},
        {key:"reasons", render:(r)=> (r.reasons||[]).join(" | ")},
        {key:"risk_flags", render:(r)=> (r.risk_flags||[]).join(" | ")},
        {key:"save_dir"},
        {key:"detail", render:(r)=> '<button class="btn" data-detail="'+fmt.esc(r.root_url)+'">View</button>', raw:true},
      ],
      tbodyId:"score-body",
      searchId:"score-search",
      pagerId:"score-pagination",
      pageSizeId:"score-page-size",
      headId:"score-head",
      initialSortKey:"score",
      initialSortDir:"desc",
    });

    const apiExclude = setupExcludeUI("api-exclude-input", "api-exclude-tags", () => {
      apiTable?.rerender();
      importantTable?.rerender();
    });
    const importantExclude = setupExcludeUI("important-exclude-input", "important-exclude-tags", () => {
      importantTable?.rerender();
    });

    const apiTable = setupTable({
      data: data.APIs || [],
      columns: [
        {key:"root_url"},
        {key:"path", render:(r)=> {
          const mark = isImportant(r.path) ? '<span class="tag" style="color:#ff9f43;border-color:rgba(255,159,67,0.4);background:rgba(255,159,67,0.08)">important</span> ' : '';
          return mark + '<span class="ellipsis" title="'+fmt.esc(r.path||"")+'">'+fmt.esc(r.path||"")+'</span>';
        }, raw:true},
        {key:"source_url"},
        {key:"save_dir"},
      ],
      tbodyId:"api-body",
      searchId:"api-search",
      pagerId:"api-pagination",
      pageSizeId:"api-page-size",
      headId:"api-head",
      extraFilter:(row)=> !matchesExcludeRow(row, apiExclude.set, ["path","root_url","source_url"]),
    });

    const importantTable = setupTable({
      data: importantAPIData,
      columns: [
        {key:"root_url"},
        {key:"path", render:(r)=> '<span class="ellipsis" title="'+fmt.esc(r.path||"")+'">'+fmt.esc(r.path||"")+'</span>', raw:true},
        {key:"source_url"},
        {key:"save_dir"},
      ],
      tbodyId:"important-body",
      searchId:"important-search",
      pagerId:"important-pagination",
      pageSizeId:"important-page-size",
      headId:"important-head",
      initialSortKey:"root_url",
      extraFilter:(row)=> !matchesExcludeRow(row, importantExclude.set, ["path","root_url","source_url"]) && !matchesExcludeRow(row, apiExclude.set, ["path","root_url","source_url"]),
    });

    setupTable({
      data: sensitiveRules,
      columns: [
        {key:"category"},
        {key:"content", render:(r)=> '<span class="ellipsis-long" title="'+fmt.esc(r.content||"")+'">'+fmt.esc(r.content||"")+'</span>', raw:true},
        {key:"source_url", render:(r)=> '<span class="ellipsis-long" title="'+fmt.esc(r.source_url||"")+'">'+fmt.esc(r.source_url||"")+'</span>', raw:true},
        {key:"entropy", render:(r)=>r.entropy?.toFixed? r.entropy.toFixed(2): r.entropy},
        {key:"save_dir"},
      ],
      tbodyId:"sens-rule-body",
      searchId:"sens-rule-search",
      pagerId:"sens-rule-pagination",
      pageSizeId:"sens-rule-page-size",
      headId:"sens-rule-head",
    });

    setupTable({
      data: sensitiveOther,
      columns: [
        {key:"category"},
        {key:"content", render:(r)=> '<span class="ellipsis-long" title="'+fmt.esc(r.content||"")+'">'+fmt.esc(r.content||"")+'</span>', raw:true},
        {key:"source_url", render:(r)=> '<span class="ellipsis-long" title="'+fmt.esc(r.source_url||"")+'">'+fmt.esc(r.source_url||"")+'</span>', raw:true},
        {key:"entropy", render:(r)=>r.entropy?.toFixed? r.entropy.toFixed(2): r.entropy},
        {key:"save_dir"},
      ],
      tbodyId:"sens-other-body",
      searchId:"sens-other-search",
      pagerId:"sens-other-pagination",
      pageSizeId:"sens-other-page-size",
      headId:"sens-other-head",
    });

    setupTable({
      data: data.SourceMaps || [],
      columns: [
        {key:"root_url"},
        {key:"js_url"},
        {key:"map_url"},
        {key:"status"},
        {key:"length"},
      ],
      tbodyId:"map-body",
      searchId:"map-search",
      pagerId:"map-pagination",
      pageSizeId:"map-page-size",
      headId:"map-head",
    });

    setupTable({
      data: data.Pages || [],
      columns: [
        {key:"root_url", render:(r)=> '<span class="ellipsis" title="'+fmt.esc(r.root_url||"")+'">'+fmt.esc(r.root_url||"")+'</span>', raw:true},
        {key:"url", render:(r)=> '<span class="ellipsis" title="'+fmt.esc(r.url||"")+'">'+fmt.esc(r.url||"")+'</span>', raw:true},
        {key:"status"},
        {key:"content_type"},
        {key:"length"},
        {key:"save_dir"},
      ],
      tbodyId:"page-body",
      searchId:"page-search",
      pagerId:"page-pagination",
      pageSizeId:"page-page-size",
      headId:"page-head",
    });

    function buildFindStats() {
      const m = {};
      const inc = (root, key) => {
        if (!root) return;
        m[root] = m[root] || {api:0,important:0,sens:0,maps:0,pages:0,cdn:0};
        m[root][key] += 1;
      };
      (data.APIs||[]).forEach(a => {
        const root = a.root_url || rootOf(a.source_url);
        inc(root, "api");
        if (isImportant(a.path)) inc(root, "important");
      });
      (data.Sensitive||[]).forEach(s => inc(rootOf(s.source_url), "sens"));
      (data.SourceMaps||[]).forEach(sm => inc(sm.root_url, "maps"));
      (data.Pages||[]).forEach(p => inc(p.root_url || rootOf(p.url), "pages"));
      (data.CDNHosts||[]).forEach(c => inc(c.Root, "cdn"));
      return m;
    }

    const detailOverlay = document.getElementById("detail-overlay");
    const detailBody = document.getElementById("detail-body");
    const detailTitle = document.getElementById("detail-title");
    const detailDrawer = document.getElementById("detail-drawer");
    document.getElementById("detail-close").onclick = () => detailOverlay.style.display = "none";

    detailOverlay.addEventListener("click", (e) => {
      if (e.target === detailOverlay) {
        detailOverlay.style.display = "none";
      }
    });
    detailDrawer.addEventListener("click", (e) => e.stopPropagation());

    document.getElementById("score-body").addEventListener("click", (e) => {
      const btn = e.target.closest("button[data-detail]");
      if (!btn) return;
      const root = btn.getAttribute("data-detail");
      renderDetail(root);
    });

    document.getElementById("summary-body").addEventListener("click", (e) => {
      const btn = e.target.closest(".mini-copy");
      if (!btn) return;
      const hash = btn.getAttribute("data-hash") || "";
      if (!hash) return;
      copyText(hash, "icon hash");
    });

    detailBody.addEventListener("click", (e) => {
      const btn = e.target.closest(".mini-copy");
      if (!btn) return;
      const hash = btn.getAttribute("data-hash") || "";
      if (!hash) return;
      copyText(hash, "icon hash");
    });


    function renderDetail(root) {
      if (!root) return;
      detailTitle.textContent = root;
      const summary = (data.Summary||[]).find(r => r.Url === root);
      const apis = (data.APIs||[]).filter(a => a.root_url === root);
      const importantAPIs = apis.filter(a => isImportant(a.path));
      const sens = (data.Sensitive||[]).filter(s => rootOf(s.source_url) === root);
      const maps = (data.SourceMaps||[]).filter(m => m.root_url === root);
      const pages = (data.Pages||[]).filter(p => p.root_url === root || rootOf(p.url) === root);
      const pageBodies = (data.PageBodies||[]).filter(p => p.root_url === root || rootOf(p.url) === root);
      const cdns = (data.CDNHosts||[]).filter(c => c.Root === root);
      const stats = findStats[root] || {api:0,important:0,sens:0,maps:0,pages:0,cdn:0};

      const iconHtml = summary && summary.IconBase64 ? '<img class="icon-thumb" src="data:image/x-icon;base64,'+fmt.esc(summary.IconBase64)+'" alt="icon" title="'+fmt.esc(summary.IconHash||"")+'">' : '<span class="small">-</span>';
      const info = summary ? ''
        + '<div class="row">Status: <code>'+fmt.esc(summary.Status)+'</code> | Icon: '+iconHtml+' <button class="mini-copy" data-hash="'+fmt.esc(summary.IconHash||"")+'">copy</button> <span class="small">'+fmt.esc(summary.IconHash || "-")+'</span></div>'
        + '<div class="row">API: '+fmt.num(summary.ApiCount||0)+' | URLs: '+fmt.num(summary.UrlCount||0)+' | CDN: '+fmt.num(summary.CDNCount||0)+'</div>'
        + '<div class="row">SaveDir: <code>'+fmt.esc(summary.SaveDir||"")+'</code></div>'
        : '<div class="row">No summary info</div>';

      const findingsSummary =
        '<div class="list-block">'
        + '<h4>Findings overview</h4>'
        + '<div class="row">API '+fmt.num(stats.api)+' · Important '+fmt.num(stats.important)+' · Sensitive '+fmt.num(stats.sens)+' · SourceMaps '+fmt.num(stats.maps)+' · Pages '+fmt.num(stats.pages)+' · CDN '+fmt.num(stats.cdn)+'</div>'
        + '</div>';

      const block = (title, arr, renderFn) => ''
        + '<div class="list-block">'
        + '<h4>'+title+' ('+arr.length+')</h4>'
        + (arr.length ? "<ul>" + arr.slice(0,200).map(renderFn).join("") + (arr.length>200 ? "<li>...more</li>":"") + "</ul>" : "<div class='small'>None</div>")
        + '</div>';

      detailBody.innerHTML = ''
        + info
        + findingsSummary
        + block("Important APIs", importantAPIs, a => "<li><code>"+fmt.esc(a.path)+"</code> <span class='small'>src "+fmt.esc(a.source_url||"")+"</span></li>")
        + block("APIs", apis, a => "<li><code>"+fmt.esc(a.path)+"</code> <span class='small'>src "+fmt.esc(a.source_url||"")+"</span></li>")
        + block("Sensitive", sens, s => "<li><code>"+fmt.esc(s.category||"")+"</code>: "+fmt.esc(s.content||"")+"</li>")
        + block("SourceMaps", maps, m => "<li><code>"+fmt.esc(m.map_url||"")+"</code> <span class='small'>js "+fmt.esc(m.js_url||"")+"</span></li>")
        + block("Pages", pages, p => "<li><code>"+fmt.esc(p.status)+"</code> "+fmt.esc(p.url||"")+" <span class='small'>"+fmt.esc(p.content_type||"")+" · "+fmt.num(p.length||0)+" bytes</span></li>")
        + block("Page snippets", pageBodies, p => "<li><code>"+fmt.esc(p.status)+"</code> "+fmt.esc(p.url||"")+"<div class='small'>"+fmt.esc((p.headers||"").slice(0,200))+"</div><div class='small'>"+fmt.esc((p.snippet||"").slice(0,400))+"</div></li>")
        + block("CDN Hosts", cdns, c => "<li><code>"+fmt.esc(c.Host||"")+"</code></li>");
      detailOverlay.style.display = "flex";
    }

    const sections = document.querySelectorAll(".section");
    const tabButtons = document.querySelectorAll(".tabs button");
    function showSection(id) {
      sections.forEach(s => s.classList.toggle("active", s.id === id));
      tabButtons.forEach(b => b.classList.toggle("active", b.dataset.target === id));
    }
    tabButtons.forEach(btn => {
      btn.addEventListener("click", () => {
        detailOverlay.style.display = "none";
        showSection(btn.dataset.target);
      });
    });
    if ((data.LLMSummary && data.LLMSummary.output) ? true : false) {
      showSection("section-llm");
    } else {
      showSection("section-score");
    }

    const tableData = {
      scores: data.Scores || [],
      summary: data.Summary || [],
      api: data.APIs || [],
      important: importantAPIData,
      sensitive_rules: sensitiveRules,
      sensitive_other: sensitiveOther,
      maps: data.SourceMaps || [],
      pages: data.Pages || [],
    };

    document.querySelectorAll("table[data-table] thead th[data-col]").forEach(th => {
      const tableEl = th.closest("table");
      const tbl = tableEl ? tableEl.dataset.table : "";
      const col = th.dataset.col;
      if (!tbl || !col) return;
      const icon = document.createElement("span");
      icon.className = "copy-icon";
      icon.textContent = "⧉";
      icon.title = "Copy column (unique)";
      icon.addEventListener("click", (e) => {
        e.stopPropagation();
        const rows = tableData[tbl] || [];
        const vals = Array.from(new Set(rows.map(r => (r && r[col]) ? String(r[col]) : "").filter(Boolean)));
        copyText(vals.join("\n"), col);
      });
      th.appendChild(icon);
    });

    function copyText(text, label) {
      if (!text) return;
      if (navigator.clipboard && navigator.clipboard.writeText) {
        navigator.clipboard.writeText(text).then(() => {
          showToast("Copied "+label+" ("+text.split(/\n/).length+" lines)");
        }).catch(() => fallbackCopy(text, label));
      } else {
        fallbackCopy(text, label);
      }
    }

    function fallbackCopy(text, label) {
      const textarea = document.createElement("textarea");
      textarea.value = text;
      document.body.appendChild(textarea);
      textarea.select();
      try { document.execCommand("copy"); showToast("Copied "+label); } catch (e) {}
      document.body.removeChild(textarea);
    }

    const toast = document.getElementById("toast");
    let toastTimer = null;
    function showToast(msg) {
      if (!toast) return;
      toast.textContent = msg;
      toast.style.display = "block";
      clearTimeout(toastTimer);
      toastTimer = setTimeout(() => { toast.style.display = "none"; }, 1800);
    }

    // graph render
    function renderGraph() {
      const svg = document.getElementById("graph-svg");
      if (!svg) return;
      const edges = [];
      const seenEdge = new Set();
      graphEdges.forEach(e => {
        const key = (e.from||"")+"->"+(e.to||"");
        if (seenEdge.has(key)) return;
        seenEdge.add(key);
        edges.push(e);
      });
      if (!edges.length) {
        svg.innerHTML = '<text x="20" y="30" fill="#8aa0c2">No graph data</text>';
        return;
      }
      const nodes = {};
      edges.forEach(e => {
        if (!nodes[e.from]) nodes[e.from] = {id:e.from, depth:e.depth, root:e.root};
        if (!nodes[e.to]) nodes[e.to] = {id:e.to, depth: Math.max(0, e.depth-1), root:e.root};
      });
      const depthGroups = {};
      Object.values(nodes).forEach(n => {
        const d = n.depth || 0;
        depthGroups[d] = depthGroups[d] || [];
        depthGroups[d].push(n);
      });
      const width = svg.clientWidth || 1200;
      const height = 640;
      const maxDepth = Math.max(...Object.keys(depthGroups).map(k => parseInt(k,10)), 1);
      const columnWidth = width / (maxDepth + 1);
      Object.entries(depthGroups).forEach(([d, arr]) => {
        arr.forEach((n, i) => {
          n.x = columnWidth * (parseInt(d,10) + 0.5);
          n.y = 40 + i * (height - 80) / Math.max(1, arr.length);
        });
      });
      function colorByStatus(n) {
        const meta = pageMeta[n.id] || {};
        const s = parseInt(meta.status,10);
        if (s >=200 && s <300) return "#5de4c7";
        if (s >=300 && s <400) return "#f9c74f";
        if (s >=400) return "#ff6b81";
        return "#8aa0c2";
      }
      function shapeByType(n) {
        const meta = pageMeta[n.id] || {};
        const t = (meta.type || "").toLowerCase();
        if (t.includes("javascript") || t.includes("css") || t.includes("image") || t.includes("font")) return "resource";
        if (n.id.endsWith(".map")) return "map";
        if (n.id.includes("/api/") || n.id.includes("/graphql")) return "api";
        return "page";
      }
      function iconPath(type, color, x, y) {
        const r = 7;
        switch (type) {
          case "resource":
            return '<rect x="'+(x-r)+'" y="'+(y-r)+'" width="'+(2*r)+'" height="'+(2*r)+'" rx="2" ry="2" fill="'+color+'" opacity="0.9"/>';
          case "api":
            return '<polygon points="'+x+','+(y-r)+' '+(x+r)+','+y+' '+x+','+(y+r)+' '+(x-r)+','+y+'" fill="'+color+'" opacity="0.9"/>';
          case "map":
            return '<line x1="'+(x-r)+'" y1="'+y+'" x2="'+(x+r)+'" y2="'+y+'" stroke="'+color+'" stroke-width="3" opacity="0.9"/><line x1="'+x+'" y1="'+(y-r)+'" x2="'+x+'" y2="'+(y+r)+'" stroke="'+color+'" stroke-width="3" opacity="0.9"/>';
          default:
            return '<circle cx="'+x+'" cy="'+y+'" r="6" fill="'+color+'" opacity="0.9"/>';
        }
      }
      let html = '';
      const stats = {page:0, resource:0, api:0, map:0};
      edges.forEach(e => {
        const from = nodes[e.from]; const to = nodes[e.to];
        if (!from || !to) return;
        html += '<line x1="'+from.x+'" y1="'+from.y+'" x2="'+to.x+'" y2="'+to.y+'" stroke="rgba(173,215,255,0.25)" stroke-width="1"/>';
      });
      Object.values(nodes).forEach(n => {
        const meta = pageMeta[n.id] || {};
        const color = colorByStatus(n);
        const tip = fmt.esc(n.id) + "\\n" + (meta.status !== undefined ? ("status: "+meta.status+" "+(meta.type||"")) : "");
        const shape = shapeByType(n);
        stats[shape] = (stats[shape]||0)+1;
        html += '<g class="graph-node" data-tip="'+tip.replace(/"/g,'&quot;')+'">'+iconPath(shape, color, n.x, n.y)+'</g>';
      });
      svg.setAttribute("height", height);
      svg.innerHTML = html;
      const statEl = document.getElementById("graph-stats");
      if (statEl) {
        statEl.innerHTML = '节点统计：page '+(stats.page||0)+' · resource '+(stats.resource||0)+' · api '+(stats.api||0)+' · map '+(stats.map||0)+' · 边 '+edges.length;
      }
      const topEl = document.getElementById("graph-top");
      if (topEl) {
        const degrees = {};
        edges.forEach(e => { degrees[e.from]=(degrees[e.from]||0)+1; });
        const top = Object.entries(degrees).sort((a,b)=>b[1]-a[1]).slice(0,5);
        topEl.innerHTML = top.length ? ('高出度节点:<br>'+top.map(([u,c])=>fmt.esc(u)+' ('+c+')').join('<br>')) : '';
      }

      const tipEl = document.getElementById("graph-tip");
      if (tipEl) {
        const showTip = (txt, x, y) => {
          tipEl.textContent = txt;
          tipEl.style.display = "block";
          tipEl.style.left = (x + 12) + "px";
          tipEl.style.top = (y + 12) + "px";
        };
        const hideTip = () => { tipEl.style.display = "none"; };
        svg.querySelectorAll(".graph-node").forEach(node => {
          node.addEventListener("mousemove", (e) => {
            const txt = node.dataset.tip || "";
            showTip(txt, e.clientX, e.clientY);
          });
          node.addEventListener("mouseleave", hideTip);
        });
      }
    }
    renderGraph();

    document.addEventListener("keydown", (e) => {
      if (e.key === "Escape") {
        detailOverlay.style.display = "none";
      }
    });
  </script>
</body>
</html>`
