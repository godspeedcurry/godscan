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
# Spider (fingerprint + API extraction)
godscan sp -u https://example.com            # aliases: sp, ss
godscan sp -f urls.txt                       # --url-file shorthand: -f or -uf

# Dir brute
godscan dir -u https://example.com           # aliases: dir, dirb, dd

# Port scan (domains/IP/CIDR/ranges)
godscan port -i '1.2.3.4/28,example.com' -p 80,443

# Icon hash
godscan icon -u https://example.com/favicon.ico

# Weak password generator
godscan weak -k "foo,bar" --full

# Clean logs
godscan clean
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

### Releases & Versioning
- Every new Git tag triggers GoReleaser (GitHub Actions) to build archives for linux/windows/macos/freebsd (amd64/arm64/386/armv7).
- Archive naming: `godscan_<version>_<os>_<arch>` (macOS labeled `macos`, windows uses zip).
- Binary version is injected from the tag; local builds without ldflags show `dev`. To embed a custom version locally: `go build -ldflags="-X github.com/godspeedcurry/godscan/cmd.version=vX.Y.Z"`.

## Development
```bash
# commit changes
git add . && git commit -m "fix bug" && git push -u origin main

# release (tag triggers GitHub Actions + GoReleaser; tag value is injected into the binary)
git tag -a v1.xx -m "v1.xx"
git push -u origin v1.xx

# delete tag (retract release)
git tag -d v1.xx
git push origin :refs/tags/v1.xx
```

## Highlights
- Web fingerprinting + API extraction: multi-probe (GET/POST/404), favicon hash (fofa/hunter), SimHash/keywords for hints.
- Crawl intelligence: DFS with depth control, stores to `spider.db`/`report.xlsx`, sensitive data + entropy surfaced, CDN/OSS profiling.
- Port/service detection: nmap-style probes plus custom JDWP/HTTP rules; supports domains, IPs, CIDRs, ranges.
- Weak passwords: base/full/mutation (incl. lunar birthday), configurable prefixes/suffixes/separators; ready for online brute or hashcat.
- Pure Go, `CGO_ENABLED=0`, ships linux/windows/macos/freebsd multi-arch builds.

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
