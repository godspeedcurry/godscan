package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var version = "v1.1.9"

func Banner() string {
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

	DefaultUA string
}

var (
	GlobalOption GlobalOptions
)

// rootCmd represents the root command
var rootCmd = &cobra.Command{
	Use:   "godscan",
	Short: Banner() + "Let's fight against the world.",
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println("godscan: try '-h' for more information")
	},
}

func Execute() {
	rootCmd.PersistentFlags().StringVarP(&GlobalOption.Url, "url", "u", "", "singel url")
	rootCmd.PersistentFlags().StringVarP(&GlobalOption.UrlFile, "url-file", "", "", "url file")

	rootCmd.PersistentFlags().StringVarP(&GlobalOption.Proxy, "proxy", "", "", "proxy")

	rootCmd.PersistentFlags().StringVarP(&GlobalOption.Host, "host", "", "", "singel host")
	rootCmd.PersistentFlags().StringVarP(&GlobalOption.HostFile, "host-file", "", "", "host file")

	rootCmd.PersistentFlags().IntVarP(&GlobalOption.LogLevel, "loglevel", "v", 2, "level of your log")
	rootCmd.PersistentFlags().StringVarP(&GlobalOption.OutputFile, "output", "o", "result.txt", "output file to write log and results")

	rootCmd.PersistentFlags().StringVarP(&GlobalOption.DefaultUA, "ua", "", "user agent", "set user agent")

	viper.BindPFlag("loglevel", rootCmd.PersistentFlags().Lookup("loglevel"))
	viper.SetDefault("loglevel", 2)

	viper.BindPFlag("DefaultUA", rootCmd.PersistentFlags().Lookup("ua"))
	viper.SetDefault("DefaultUA", "Mozilla/5.0 (Windows NT 11.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/114.0.5773.219 Safari/537.36 Edg/109.0.1711.53")

	viper.BindPFlag("proxy", rootCmd.PersistentFlags().Lookup("proxy"))
	viper.SetDefault("proxy", "")

	cobra.CheckErr(rootCmd.Execute())
}
