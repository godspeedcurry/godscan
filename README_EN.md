# godscan
Focused on API discovery, sensitive data surfacing, and weak/strong password generation.

## Focus
- API detection: multi-probe fingerprinting + JS/Vue parsing with de-dup, persisted to `spider.db` / `report.xlsx`.
- Sensitive data: HTML/JS scan with entropy highlighting, SourceMap parsing, saved into `spider.db` and `sourcemaps.txt`.
- Passwords: keyword combos, mutations, lunar-birthday variants for direct brute/dict use.

## Highlights
- Fingerprinting + API extraction: GET/POST/404 probes, favicon hash (fofa/hunter), SimHash/keywords.
- Crawl intel: DFS depth control, stores to `spider.db`/`report.xlsx`, sensitive/entropy/CDN/OSS in one run.
- Source map discovery: auto-probe same-origin `.map` from scripts/JS bodies with minimal requests.
- Homepage snapshots: store home HTML/headers for fofa-style body/header searches.
- Port/service detection: nmap-style probes plus custom HTTP/JDWP rules.
- Password generator: base/mutation/lunar, configurable prefixes/suffixes/separators.
- Pure Go, `CGO_ENABLED=0`, multi-platform builds.

## Quick Usage
```bash
# Spider (fingerprint + API + sensitive)
godscan sp -u https://example.com            # aliases: sp, ss
godscan sp -f urls.txt                       # supports -f/-uf

# SourceMap / sensitive / homepage search
godscan grep "js.map"
godscan grep "elastic"   # body/header

# Dir / port / weak passwords
godscan dir -u https://example.com
godscan port -i '1.2.3.4/28,example.com' -p 80,443
godscan weak -k "foo,bar" --full

# Export offline HTML report (large tables with paging/search)
godscan report --html report.html
```

## Output
- `spider.db` / `report.xlsx`: fingerprint, APIs, sensitive hits, CDN, SourceMap, homepage snapshots.
- `output/`: `result.log`, `spider_summary.json`, `sourcemaps.txt`, etc.

## Recent updates
- Source map probing: same-origin `.map` via HEAD-first, saved into DB with fewer requests.
- Search: `grep` queries api/sensitive/map/page by default; no table flag needed for SourceMap/homepage/body/header.
- Homepage snapshots added; README focused on API/sensitive/password use cases.
