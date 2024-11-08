package cmd

import (
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/fatih/color"
	"github.com/godspeedcurry/godscan/utils"
	"github.com/google/go-github/v57/github"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var version = "v1.1.27"

func checkForUpdate(currentVersion string) {
	ctx := context.Background()
	client := github.NewClient(nil)
	release, _, err := client.Repositories.GetLatestRelease(ctx, "godspeedcurry", "godscan")
	if err != nil {
		fmt.Println("Error checking for updates:", err)
		return
	}

	if release.TagName != nil && *release.TagName != currentVersion {
		fmt.Println(color.RedString("Update available: %s. Please download the latest version from %s", *release.TagName, *release.HTMLURL))
	}
}
func Banner() string {
	checkForUpdate(version)
	banner := `
██████╗   ██████╗ ██████╗ ███████╗ ██████╗ █████╗ ███╗   ██╗
██╔════╝ ██╔═══██╗██╔══██╗██╔════╝██╔════╝██╔══██╗████╗  ██║
██║  ███╗██║   ██║██║  ██║███████╗██║     ███████║██╔██╗ ██║
██║   ██║██║   ██║██║  ██║╚════██║██║     ██╔══██║██║╚██╗██║
╚██████╔╝╚██████╔╝██████╔╝███████║╚██████╗██║  ██║██║ ╚████║
 ╚═════╝  ╚═════╝ ╚═════╝ ╚══════╝ ╚═════╝╚═╝  ╚═╝╚═╝  ╚═══╝															
godscan version: ` + version + `
`
	return banner
}

type GlobalOptions struct {
	Url     string
	UrlFile string

	Host     string
	HostFile string

	LogLevel   int
	OutputFile string
	Proxy      string

	DefaultUA     string
	ScanPrivateIp bool
	Headers       []string
}

var (
	GlobalOption GlobalOptions
)

// rootCmd represents the root command
var rootCmd = &cobra.Command{
	Use:   "godscan",
	Short: Banner() + "Let's fight against the world.",
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
	rootCmd.PersistentFlags().StringVarP(&GlobalOption.Url, "url", "u", "", "singel url")
	rootCmd.PersistentFlags().StringVarP(&GlobalOption.UrlFile, "url-file", "", "", "url file")

	rootCmd.PersistentFlags().StringVarP(&GlobalOption.Proxy, "proxy", "", "", "proxy")

	rootCmd.PersistentFlags().StringVarP(&GlobalOption.Host, "host", "", "", "singel host")
	rootCmd.PersistentFlags().StringVarP(&GlobalOption.HostFile, "host-file", "", "", "host file")

	rootCmd.PersistentFlags().IntVarP(&GlobalOption.LogLevel, "loglevel", "v", 2, "level of your log")
	rootCmd.PersistentFlags().StringVarP(&GlobalOption.OutputFile, "output", "o", "result.log", "output file to write log and results")

	rootCmd.PersistentFlags().StringVarP(&GlobalOption.DefaultUA, "ua", "", "user agent", "set user agent")

	rootCmd.PersistentFlags().BoolVarP(&GlobalOption.ScanPrivateIp, "private-ip", "", false, "scan private ip")

	rootCmd.PersistentFlags().StringArrayVarP(&GlobalOption.Headers, "headers", "H", nil, "Custom headers, eg: -H 'Cookie: 123'")

	viper.BindPFlag("loglevel", rootCmd.PersistentFlags().Lookup("loglevel"))
	viper.SetDefault("loglevel", 2)

	viper.BindPFlag("DefaultUA", rootCmd.PersistentFlags().Lookup("ua"))
	viper.SetDefault("DefaultUA", "Mozilla/5.0 (Windows NT 11.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/114.0.5773.219 Safari/537.36 Edg/109.0.1711.53")

	viper.BindPFlag("proxy", rootCmd.PersistentFlags().Lookup("proxy"))
	viper.SetDefault("proxy", SetProxyFromEnv())

	viper.BindPFlag("output", rootCmd.PersistentFlags().Lookup("output"))
	viper.SetDefault("output", "result.log")

	viper.BindPFlag("private-ip", rootCmd.PersistentFlags().Lookup("private-ip"))
	viper.SetDefault("private-ip", false)

	viper.BindPFlag("headers", rootCmd.PersistentFlags().Lookup("headers"))
	viper.SetDefault("headers", []string{})

}

func Execute() {
	if viper.GetString("proxy") != "" {
		fmt.Fprintf(os.Stderr, "[%s] [%s] Proxy is %s\n", utils.GetCurrentTime(), color.New(color.FgCyan).Sprintf("%s", "INFO"), viper.GetString("proxy"))
	} else {
		fmt.Fprintf(os.Stderr, "[%s] [%s] Proxy is null\n", utils.GetCurrentTime(), color.New(color.FgCyan).Sprintf("%s", "INFO"))
	}
	cobra.CheckErr(rootCmd.Execute())
}
