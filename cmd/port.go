/*
Copyright © 2023 NAME HERE <EMAIL ADDRESS>
*/
package cmd

import (
	"fmt"
	"strings"

	"github.com/godspeedcurry/godscan/common"
	"github.com/godspeedcurry/godscan/utils"
	"github.com/spf13/viper"
)

type PortOptions struct {
	IpRange         string
	IpRangeFile     string
	PortRange       string
	TopPorts        int
	useAllProbes    bool
	nullProbeOnly   bool
	scanSendTimeout int
	scanReadTimeout int
	scanRarity      int
	Threads         int
}

var (
	portOptions PortOptions
)

func (o *PortOptions) validateOptions() error {
	if portOptions.IpRange == "" && portOptions.IpRangeFile == "" {
		return fmt.Errorf("please give ips")
	}
	return nil
}
func init() {
	ipCmd := newCommandWithAliases("port", "port scanner", []string{"pp"}, &portOptions)
	rootCmd.AddCommand(ipCmd)

	ipCmd.PersistentFlags().StringVarP(&portOptions.IpRange, "host", "i", "", "your ip list")
	ipCmd.PersistentFlags().StringVarP(&portOptions.IpRangeFile, "host-file", "I", "", "your ip list file")

	ipCmd.PersistentFlags().StringVarP(&portOptions.PortRange, "port", "p", strings.Join(common.DefaultPorts, ","), "your port list")
	ipCmd.PersistentFlags().IntVarP(&portOptions.TopPorts, "top", "", 0, "top ports to scan, default is 500")

	ipCmd.PersistentFlags().IntVarP(&portOptions.scanSendTimeout, "scan-send-timeout", "s", 5, "Set connection send timeout in seconds")
	ipCmd.PersistentFlags().IntVarP(&portOptions.scanReadTimeout, "scan-read-timeout", "r", 5, "Set connection read timeout in seconds")
	ipCmd.PersistentFlags().IntVarP(&portOptions.scanRarity, "scan-rarity", "R", 7, "Scan Rarity")

	ipCmd.PersistentFlags().BoolVarP(&portOptions.useAllProbes, "all-probe", "a", false, "Use all probes to probe service")
	ipCmd.PersistentFlags().BoolVarP(&portOptions.nullProbeOnly, "null-probe-only", "n", false, "Use all probes to probe service")

	ipCmd.PersistentFlags().IntVarP(&portOptions.Threads, "threads", "t", 1000, "Number of threads to use, default is 1000")

	viper.BindPFlag("host", ipCmd.PersistentFlags().Lookup("host"))
	viper.SetDefault("host", "")

	viper.BindPFlag("threads", ipCmd.PersistentFlags().Lookup("threads"))
	viper.SetDefault("threads", 1000)

	viper.BindPFlag("port", ipCmd.PersistentFlags().Lookup("port"))
	viper.SetDefault("port", "")

	viper.BindPFlag("top", ipCmd.PersistentFlags().Lookup("top"))
	viper.SetDefault("top", 0)

	viper.BindPFlag("scan-rarity", ipCmd.PersistentFlags().Lookup("scan-rarity"))
	viper.SetDefault("scan-rarity", 5)

	viper.BindPFlag("scan-send-timeout", ipCmd.PersistentFlags().Lookup("scan-send-timeout"))
	viper.SetDefault("scan-send-timeout", 5)

	viper.BindPFlag("scan-read-timeout", ipCmd.PersistentFlags().Lookup("scan-read-timeout"))
	viper.SetDefault("scan-read-timeout", 5)

	viper.BindPFlag("null-probe-only", ipCmd.PersistentFlags().Lookup("null-probe-only"))
	viper.SetDefault("null-probe-only", false)

	viper.BindPFlag("all-probe", ipCmd.PersistentFlags().Lookup("all-probe"))
	viper.SetDefault("all-probe", false)
}

func (o *PortOptions) run() {
	if portOptions.IpRangeFile != "" {
		ips := utils.FileReadLine(portOptions.IpRangeFile)
		portOptions.IpRange = strings.Join(ips, ",")
	} else if portOptions.IpRange == "" {
		utils.Info("Please provide ip range or ip range file")
		return
	}
	if portOptions.TopPorts != 0 {
		if portOptions.TopPorts > 20000 {
			utils.Info("We do not have more than top 20000 ports, please choose a smaller number, or just scan all ports use `-p 0-65535`")
			return
		} else {
			utils.PortScan(portOptions.IpRange, strings.Join(strings.Split(common.AllPorts, ",")[0:portOptions.TopPorts], ","))
		}

	} else {
		utils.PortScan(portOptions.IpRange, portOptions.PortRange)
	}

}
