package cmd

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"github.com/godspeedcurry/godscan/utils"
	"github.com/google/go-github/v57/github"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

// version is injected at build time via -ldflags "-X github.com/godspeedcurry/godscan/cmd.version=vX.Y.Z".
// Default to "dev" for local builds.
var version = "dev"
var updateCheckOnce sync.Once

func checkForUpdate(currentVersion string) {
	ctx := context.Background()
	client := github.NewClient(nil)
	release, _, err := client.Repositories.GetLatestRelease(ctx, "godspeedcurry", "godscan")
	if err != nil {
		utils.Info("Error checking for updates: %v", err)
		return
	}

	if release.TagName != nil && *release.TagName != currentVersion {
		utils.Warning("Update available: %s. Please download the latest version from %s", *release.TagName, *release.HTMLURL)
	}
}

func bannerText() string {
	return `
██████╗   ██████╗ ██████╗ ███████╗ ██████╗ █████╗ ███╗   ██╗
██╔════╝ ██╔═══██╗██╔══██╗██╔════╝██╔════╝██╔══██╗████╗  ██║
██║  ███╗██║   ██║██║  ██║███████╗██║     ███████║██╔██╗ ██║
██║   ██║██║   ██║██║  ██║╚════██║██║     ██╔══██║██║╚██╗██║
╚██████╔╝╚██████╔╝██████╔╝███████║╚██████╗██║  ██║██║ ╚████║
 ╚═════╝  ╚═════╝ ╚═════╝ ╚══════╝ ╚═════╝╚═╝  ╚═╝╚═╝  ╚═══╝															
godscan version: ` + version + `
`
}

func maybeCheckForUpdate() {
	updateCheckOnce.Do(func() { checkForUpdate(version) })
}

type GlobalOptions struct {
	Url     string
	UrlFile string

	Host     string
	HostFile string

	LogLevel   int
	OutputFile string
	OutputDir  string
	Proxy      string

	DefaultUA     string
	ScanPrivateIp bool
	Headers       []string

	Filter []string
}

var (
	GlobalOption GlobalOptions
)

var rootCmd = &cobra.Command{
	Use:   "godscan",
	Short: bannerText() + "Let's fight against the world.",
	Long: `godscan - web recon, API extraction, weakpass generator, port fingerprint.

Quick usage:
  godscan sp -u https://example.com          # spider (alias: sp, ss)
  godscan sp -f urls.txt                     # spider from file (alias: --url-file, -f, -uf)
  godscan dir -u https://example.com         # dirbrute (alias: dirb, dd)
  godscan port -i 1.2.3.4/28 -p 80,443       # port scan (alias: pp)
  godscan icon -u https://example.com/ico.ico
  godscan weak -k "foo,bar"                  # weakpass generator

Global flags (common):
  -u, --url            single target URL
  -f, --url-file       file with URLs (also accepts -uf file)
      --host / --host-file   IP/domain list for port scan
  -e, --filter         filter substrings
  -H, --headers        custom headers
  -o, --output         log/result file (default result.log)
      --proxy          HTTP(S)/SOCKS proxy
      --private-ip     include private IP ranges
      --ua             custom User-Agent
      --loglevel       0=minimal, 2=default, 6=debug`,
	PersistentPreRun: func(cmd *cobra.Command, args []string) {
		maybeCheckForUpdate()
	},
	Run: func(cmd *cobra.Command, args []string) {
		cmd.Help()
	},
}

type CommandOptions interface {
	validateOptions() error
	run()
}

func newCommandWithAliases(use, shortDesc string, aliases []string, opts CommandOptions) *cobra.Command {
	return &cobra.Command{
		Use:     use,
		Short:   fmt.Sprintf("%s (Aliases: %s)", shortDesc, strings.Join(aliases, ", ")),
		Aliases: aliases,
		Run: func(cmd *cobra.Command, args []string) {
			if err := opts.validateOptions(); err != nil {
				cmd.Help()
				os.Exit(0)
			}
			opts.run()

		},
	}
}

func SetProxyFromEnv() string {
	proxyEnvVars := []string{"HTTP_PROXY", "http_proxy", "HTTPS_PROXY", "https_proxy", "ALL_PROXY", "all_proxy"}
	for _, envVar := range proxyEnvVars {
		if val := os.Getenv(envVar); val != "" {
			return val
		}
	}
	return ""
}

func init() {
	rootCmd.PersistentFlags().StringVarP(&GlobalOption.Url, "url", "u", "", "single target URL")
	rootCmd.PersistentFlags().StringVarP(&GlobalOption.UrlFile, "url-file", "f", "", "short alias, -uf, file with target URLs (one per line)")

	rootCmd.PersistentFlags().StringVarP(&GlobalOption.Proxy, "proxy", "", "", "http(s)/socks proxy, e.g. http://127.0.0.1:8080")

	rootCmd.PersistentFlags().StringVarP(&GlobalOption.Host, "host", "", "", "single host or CIDR (for port scan)")
	rootCmd.PersistentFlags().StringVarP(&GlobalOption.HostFile, "host-file", "", "", "file with host entries (one per line)")

	rootCmd.PersistentFlags().IntVarP(&GlobalOption.LogLevel, "loglevel", "v", 2, "log verbosity for info/debug (warnings/errors always shown). 0=minimal, 2=default, 6=debug")
	rootCmd.PersistentFlags().StringVarP(&GlobalOption.OutputFile, "output", "o", "result.log", "file to write logs and results")
	rootCmd.PersistentFlags().StringVarP(&GlobalOption.OutputDir, "output-dir", "O", "output", "directory for logs/results")

	rootCmd.PersistentFlags().StringVarP(&GlobalOption.DefaultUA, "ua", "", "user agent", "override default User-Agent header")

	rootCmd.PersistentFlags().BoolVarP(&GlobalOption.ScanPrivateIp, "private-ip", "", false, "scan private ip")

	rootCmd.PersistentFlags().StringArrayVarP(&GlobalOption.Headers, "headers", "H", nil, "Custom headers, eg: -H 'Cookie: 123'")

	rootCmd.PersistentFlags().StringArrayVarP(&GlobalOption.Filter, "filter", "e", nil, "Filter url, eg: -e 'abc.com'")

	viper.BindPFlag("loglevel", rootCmd.PersistentFlags().Lookup("loglevel"))
	viper.SetDefault("loglevel", 2)

	viper.BindPFlag("DefaultUA", rootCmd.PersistentFlags().Lookup("ua"))
	viper.SetDefault("DefaultUA", "Mozilla/5.0 (Windows NT 11.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/114.0.5773.219 Safari/537.36 Edg/109.0.1711.53")

	viper.BindPFlag("proxy", rootCmd.PersistentFlags().Lookup("proxy"))
	viper.SetDefault("proxy", SetProxyFromEnv())

	viper.BindPFlag("output", rootCmd.PersistentFlags().Lookup("output"))
	viper.SetDefault("output", "result.log")
	viper.BindPFlag("output-dir", rootCmd.PersistentFlags().Lookup("output-dir"))
	viper.SetDefault("output-dir", "output")

	viper.BindPFlag("private-ip", rootCmd.PersistentFlags().Lookup("private-ip"))
	viper.SetDefault("private-ip", false)

	viper.BindPFlag("headers", rootCmd.PersistentFlags().Lookup("headers"))
	viper.SetDefault("headers", []string{})

	viper.BindPFlag("filter", rootCmd.PersistentFlags().Lookup("filter"))
	viper.SetDefault("filter", []string{})

	rootCmd.PersistentFlags().Bool("json", false, "enable json log output")
	rootCmd.PersistentFlags().Bool("quiet", false, "suppress console output")
	rootCmd.PersistentFlags().Bool("insecure", true, "skip TLS verification")
	rootCmd.PersistentFlags().Int("http-timeout", 10, "http client timeout (seconds)")
	rootCmd.PersistentFlags().Int("conn-per-host", 0, "max connections per host (0=auto)")
	rootCmd.PersistentFlags().Int("max-body-bytes", 2*1024*1024, "max response body bytes to read")

	viper.BindPFlag("json", rootCmd.PersistentFlags().Lookup("json"))
	viper.SetDefault("json", false)
	viper.BindPFlag("quiet", rootCmd.PersistentFlags().Lookup("quiet"))
	viper.SetDefault("quiet", false)
	viper.BindPFlag("insecure", rootCmd.PersistentFlags().Lookup("insecure"))
	viper.SetDefault("insecure", true)
	viper.BindPFlag("http-timeout", rootCmd.PersistentFlags().Lookup("http-timeout"))
	viper.SetDefault("http-timeout", 10)
	viper.BindPFlag("conn-per-host", rootCmd.PersistentFlags().Lookup("conn-per-host"))
	viper.SetDefault("conn-per-host", 0)
	viper.BindPFlag("max-body-bytes", rootCmd.PersistentFlags().Lookup("max-body-bytes"))
	viper.SetDefault("max-body-bytes", 2*1024*1024)
}

func Execute() {
	os.Args = expandUF(os.Args)
	normalizeOutputPaths()
	if viper.GetString("proxy") != "" {
		utils.Info("Proxy is %s", viper.GetString("proxy"))
	} else {
		utils.Info("Proxy is not set")
	}
	cobra.CheckErr(rootCmd.Execute())
}

// normalizeOutputPaths ensures output dir exists and output file is placed under it unless an absolute path is provided.
func normalizeOutputPaths() {
	dir := viper.GetString("output-dir")
	if dir == "" {
		dir = "output"
	}
	if err := os.MkdirAll(dir, 0755); err != nil {
		utils.Warning("create output dir failed: %v", err)
	}
	out := viper.GetString("output")
	if out == "" {
		out = "result.log"
	}
	if !filepath.IsAbs(out) {
		out = filepath.Join(dir, filepath.Base(out))
	}
	viper.Set("output-dir", dir)
	viper.Set("output", out)
}

// expandUF allows fscan-style "-uf file.txt" or "-uf=file.txt" as an alias for "-f file.txt".
// Cobra does not support multi-letter shorthands, so we normalize here before flag parsing.
func expandUF(args []string) []string {
	out := make([]string, 0, len(args))
	for i := 0; i < len(args); i++ {
		a := args[i]
		switch {
		case a == "-uf":
			if i+1 < len(args) {
				out = append(out, "-f", args[i+1])
				i++
			} else {
				out = append(out, "-f")
			}
		case strings.HasPrefix(a, "-uf="):
			out = append(out, "-f", strings.TrimPrefix(a, "-uf="))
		case strings.HasPrefix(a, "-uf"):
			// e.g., -uffile.txt
			out = append(out, "-f", strings.TrimPrefix(a, "-uf"))
		default:
			out = append(out, a)
		}
	}
	return out
}
