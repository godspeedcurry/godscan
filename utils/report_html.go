package utils

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"html/template"
	"net/url"
	"os"
	"time"
)

type HTMLReportData struct {
	GeneratedAt string
	Summary     []SpiderRecord
	APIs        []APIPathRow
	Sensitive   []SensitiveHit
	SourceMaps  []SourceMapHit
	Pages       []PageSnapshotMeta
	PageBodies  []PageSnapshotLite
	Scores      []ScoredRow
	CDNHosts    []CDNHostRow
}

type ScoredRow struct {
	RootURL   string   `json:"root_url"`
	Score     int      `json:"score"`
	Status    int      `json:"status"`
	ApiCount  int      `json:"api_count"`
	UrlCount  int      `json:"url_count"`
	CDNCount  int      `json:"cdn_count"`
	SaveDir   string   `json:"save_dir"`
	Reasons   []string `json:"reasons"`
	RiskFlags []string `json:"risk_flags"`
}

func ExportHTMLReport(db *sql.DB, outputPath string) error {
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

	data := HTMLReportData{
		GeneratedAt: time.Now().Format(time.RFC3339),
		Summary:     summary,
		APIs:        apis,
		Sensitive:   sens,
		SourceMaps:  smaps,
		Pages:       pages,
		PageBodies:  pageBodies,
		CDNHosts:    cdns,
	}
	data.Scores = buildScores(data)
	return renderHTMLReport(outputPath, data)
}

func buildScores(data HTMLReportData) []ScoredRow {
	mapCounts := make(map[string]int)
	for _, m := range data.SourceMaps {
		mapCounts[m.RootURL]++
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
			UrlCount:  rec.UrlCount,
			CDNCount:  rec.CDNCount,
			SaveDir:   rec.SaveDir,
			Reasons:   reasons,
			RiskFlags: risks,
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
    table { width: 100%; border-collapse: collapse; }
    th, td { padding: 8px 10px; border-bottom: 1px solid rgba(255,255,255,0.05); font-size: 12px; }
    .ellipsis { max-width: 520px; display: inline-block; white-space: nowrap; overflow: hidden; text-overflow: ellipsis; vertical-align: bottom; }
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
    .drawer-overlay { position: fixed; inset: 0; background: rgba(0,0,0,0.55); backdrop-filter: blur(4px); display: none; align-items: center; justify-content: center; z-index: 20; }
    .drawer { width: min(960px, 95vw); max-height: 90vh; overflow: auto; background: var(--panel); border: 1px solid rgba(255,255,255,0.08); border-radius: 14px; padding: 16px; box-shadow: 0 12px 60px rgba(0,0,0,0.4); }
    .drawer h3 { margin: 0 0 8px 0; color: var(--accent-2); }
    .drawer .row { margin-bottom: 8px; color: var(--muted); font-size: 13px; }
    .drawer code { background: rgba(255,255,255,0.05); padding: 2px 4px; border-radius: 4px; }
    .list-block { margin: 8px 0; padding: 8px; border: 1px solid rgba(255,255,255,0.05); border-radius: 8px; background: rgba(255,255,255,0.03); }
    .list-block h4 { margin: 0 0 6px 0; font-size: 13px; color: var(--accent); }
  </style>
</head>
<body>
  <header>
    <h1>godscan HTML report</h1>
    <div class="pill">Generated at <span id="generated-at"></span></div>
  </header>
  <nav class="tabs">
    <button class="active" data-target="section-score">Scores</button>
    <button data-target="section-summary">Summary</button>
    <button data-target="section-api">APIs</button>
    <button data-target="section-sensitive">Sensitive</button>
    <button data-target="section-maps">SourceMaps</button>
    <button data-target="section-pages">Pages</button>
  </nav>
  <main>
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
        <table>
          <thead id="score-head"><tr><th data-col="root_url">Root</th><th data-col="score">Score</th><th data-col="status">Status</th><th data-col="api_count">API</th><th data-col="url_count">URLs</th><th data-col="cdn_count">CDN</th><th data-col="reasons">Reasons</th><th data-col="risk_flags">Risk</th><th data-col="save_dir">Save Dir</th><th>Detail</th></tr></thead>
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
        <table>
          <thead id="summary-head">
            <tr>
              <th data-col="Url">URL</th><th data-col="IconHash">Icon hash</th><th data-col="ApiCount">API</th><th data-col="UrlCount">URLs</th><th data-col="CDNCount">CDN URLs</th><th data-col="CDNHosts">CDN Hosts</th><th data-col="Status">Status</th><th data-col="SaveDir">Save Dir</th>
            </tr>
          </thead>
          <tbody id="summary-body"></tbody>
        </table>
      </div>
    </section>

    {{/* APIs */}}
    <section class="panel section" id="section-api">
      <header>
        <h2>APIs</h2>
        <div class="controls">
          <input id="api-search" type="search" placeholder="Filter root/path/source">
          <select id="api-page-size">
            <option value="200">200 / page</option>
            <option value="500">500 / page</option>
            <option value="1000">1000 / page</option>
          </select>
          <div class="pagination" id="api-pagination"></div>
        </div>
      </header>
      <div style="overflow:auto">
        <table>
          <thead id="api-head"><tr><th data-col="root_url">Root</th><th data-col="path">Path</th><th data-col="source_url">Source</th><th data-col="save_dir">Save Dir</th></tr></thead>
          <tbody id="api-body"></tbody>
        </table>
      </div>
    </section>

    {{/* Sensitive */}}
    <section class="panel section" id="section-sensitive">
      <header>
        <h2>Sensitive</h2>
        <div class="controls">
          <input id="sens-search" type="search" placeholder="Filter category/content/source">
          <select id="sens-page-size">
            <option value="200">200 / page</option>
            <option value="500">500 / page</option>
            <option value="1000">1000 / page</option>
          </select>
          <div class="pagination" id="sens-pagination"></div>
        </div>
      </header>
      <div style="overflow:auto">
        <table>
          <thead id="sens-head"><tr><th data-col="category">Category</th><th data-col="content">Content</th><th data-col="source_url">Source</th><th data-col="entropy">Entropy</th><th data-col="save_dir">Save Dir</th></tr></thead>
          <tbody id="sens-body"></tbody>
        </table>
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
        <table>
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
        <table>
          <thead id="page-head"><tr><th data-col="root_url">Root</th><th data-col="url">URL</th><th data-col="status">Status</th><th data-col="content_type">Type</th><th data-col="length">Length</th><th data-col="save_dir">Save Dir</th></tr></thead>
          <tbody id="page-body"></tbody>
        </table>
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

  <script>
    const data = {{.DataJSON}};
    const fmt = {
      esc: (s) => String(s || "").replace(/[&<>"']/g, c => ({'&':'&amp;','<':'&lt;','>':'&gt;','"':'&quot;',"'":'&#39;'}[c])),
      num: (n) => isFinite(n) ? n.toLocaleString() : n
    };

    function rootOf(u) {
      try { const x = new URL(u); return x.protocol + "//" + x.host; } catch { return ""; }
    }

    function setupTable({data, columns, tbodyId, searchId, pagerId, pageSizeId, headId, defaultSize=200, initialSortKey=null, initialSortDir="asc"}) {
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
        if (!filter) return data;
        const f = filter.toLowerCase();
        return data.filter(row => columns.some(col => (row[col.key] || "").toString().toLowerCase().includes(f)));
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
        {key:"IconHash"},
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
        {key:"url_count"},
        {key:"cdn_count"},
        {key:"findings", render:(r)=> {
          const s = findStats[r.root_url] || {};
          return "API:"+ (s.api||0)+" / Sensitive:"+ (s.sens||0)+" / Maps:"+ (s.maps||0)+" / Pages:"+ (s.pages||0)+" / CDN:"+ (s.cdn||0);
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

    setupTable({
      data: data.APIs || [],
      columns: [
        {key:"root_url"},
        {key:"path"},
        {key:"source_url"},
        {key:"save_dir"},
      ],
      tbodyId:"api-body",
      searchId:"api-search",
      pagerId:"api-pagination",
      pageSizeId:"api-page-size",
      headId:"api-head",
    });

    setupTable({
      data: data.Sensitive || [],
      columns: [
        {key:"category"},
        {key:"content"},
        {key:"source_url"},
        {key:"entropy", render:(r)=>r.entropy?.toFixed? r.entropy.toFixed(2): r.entropy},
        {key:"save_dir"},
      ],
      tbodyId:"sens-body",
      searchId:"sens-search",
      pagerId:"sens-pagination",
      pageSizeId:"sens-page-size",
      headId:"sens-head",
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
        m[root] = m[root] || {api:0,sens:0,maps:0,pages:0,cdn:0};
        m[root][key] += 1;
      };
      (data.APIs||[]).forEach(a => inc(a.root_url || rootOf(a.source_url), "api"));
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

    function renderDetail(root) {
      if (!root) return;
      detailTitle.textContent = root;
      const summary = (data.Summary||[]).find(r => r.Url === root);
      const apis = (data.APIs||[]).filter(a => a.root_url === root);
      const sens = (data.Sensitive||[]).filter(s => rootOf(s.source_url) === root);
      const maps = (data.SourceMaps||[]).filter(m => m.root_url === root);
      const pages = (data.Pages||[]).filter(p => p.root_url === root || rootOf(p.url) === root);
      const pageBodies = (data.PageBodies||[]).filter(p => p.root_url === root || rootOf(p.url) === root);
      const cdns = (data.CDNHosts||[]).filter(c => c.Root === root);
      const stats = findStats[root] || {api:0,sens:0,maps:0,pages:0,cdn:0};

      const info = summary ? ''
        + '<div class="row">Status: <code>'+fmt.esc(summary.Status)+'</code> | Icon: <code>'+fmt.esc(summary.IconHash || "-")+'</code></div>'
        + '<div class="row">API: '+fmt.num(summary.ApiCount||0)+' | URLs: '+fmt.num(summary.UrlCount||0)+' | CDN: '+fmt.num(summary.CDNCount||0)+'</div>'
        + '<div class="row">SaveDir: <code>'+fmt.esc(summary.SaveDir||"")+'</code></div>'
        : '<div class="row">No summary info</div>';

      const findingsSummary =
        '<div class="list-block">'
        + '<h4>Findings overview</h4>'
        + '<div class="row">API '+fmt.num(stats.api)+' · Sensitive '+fmt.num(stats.sens)+' · SourceMaps '+fmt.num(stats.maps)+' · Pages '+fmt.num(stats.pages)+' · CDN '+fmt.num(stats.cdn)+'</div>'
        + '</div>';

      const block = (title, arr, renderFn) => ''
        + '<div class="list-block">'
        + '<h4>'+title+' ('+arr.length+')</h4>'
        + (arr.length ? "<ul>" + arr.slice(0,200).map(renderFn).join("") + (arr.length>200 ? "<li>...more</li>":"") + "</ul>" : "<div class='small'>None</div>")
        + '</div>';

      detailBody.innerHTML = ''
        + info
        + findingsSummary
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
      btn.addEventListener("click", () => showSection(btn.dataset.target));
    });
    showSection("section-score");
  </script>
</body>
</html>`
