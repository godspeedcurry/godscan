# godscan
<h4 align="center">Your next scanner doesn’t have to be a scanner.</h4>

<p align="center">
  <a href="https://goreportcard.com/report/github.com/godspeedcurry/godscan">
    <img src="https://goreportcard.com/badge/github.com/godspeedcurry/godscan">	
  </a>
  <a href="https://opensource.org/licenses/MIT">
    <img src="https://img.shields.io/badge/license-MIT-_red.svg">
  </a>
  <a href="https://github.com/godspeedcurry/godscan/releases">
  	<img src="https://img.shields.io/github/downloads/godspeedcurry/godscan/total">
  </a>
</p>


## Quick Usage

```bash
# One target
./godscan dirbrute --url http://www.example.com
# Batch
./godscan dirbrute --url-file url.txt

# Icon hash (fofa/hunter)
./godscan icon --url http://www.example.com/favicon.ico

# Port scan (top 500 by default)
./godscan port -i '1.2.3.4/28'
# Custom ports
./godscan port -i '1.2.3.4/28' -p '12312-12334,6379,22'
# Domains are supported (auto-resolve)
# ./godscan port -i 'example.com,foo.bar'

# Weak password generator
./godscan weakpass -k "ZhangSan,110101199003070759,18288888888"
./godscan weakpass -k "baidu,admin,root,server" --full > dict.txt

# Spider + API/fingerprint
./godscan spider --url http://example.com
# Depth defaults to 2; increase with -d
./godscan spider --url-file url.txt

# Clean logs
./godscan clean
```

### Flags (common)
- `-u, --url string` single target URL
- `--url-file string` file with URLs (one per line)
- `-e, --filter stringArray` substring filters to drop URLs
- `--private-ip` include private IP ranges (off by default)
- `-o, --output string` log/result file (default `result.log`)
- `-v, --loglevel int` log level (default 2)
- `--proxy string` HTTP proxy

### Notable Features
- **Dirbrute**: Small curated wordlist focusing on high-value leaks and code-exec spots.
- **Fingerprinting**: Multi-probe (GET/POST/404), favicon hash (fofa/hunter), similarity hash, keyword hints.
- **Spider**: DFS crawl with API path extraction from JS (Vue chunk detection included); saves per-target artifacts to `YYYY-MM-DD/<host>_port/spider/`.
- **Sensitive data hunt**: Regex + entropy to surface secrets, URLs, tokens; stored in SQLite (`spider.db`) and `report.xlsx`.
- **Weakpass**: Generates focused weak-password lists; supports prefixes/suffixes/separators and `--full` mutation mode.
- **Port scan**: Nmap-style probes plus custom JDWP/HTTP tweaks; configurable top lists and ranges.

### Autocomplete
```bash
./godscan completion zsh > /tmp/x
source /tmp/x
```

### Reports
- Spider results persist to `spider.db`; run `godscan report` to export `report.xlsx`.
- `godscan clean` removes logs.

### Cross-Compilation
```bash
CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -ldflags="-s -w" -trimpath -o godscan_linux_amd64
CGO_ENABLED=0 GOOS=windows GOARCH=amd64 go build -ldflags="-s -w" -trimpath -o godscan_win_amd64
CGO_ENABLED=0 GOOS=darwin GOARCH=amd64 go build -ldflags="-s -w" -trimpath -o godscan_darwin_amd64
```

For the full Chinese documentation, see `README.md`.
