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

type WeakPassOptions struct {
	Keywords   string
	Suffix     string
	Separator  string
	Prefix     string
	Full       bool
	ListFormat bool
	Variant    bool
	Show       bool
}

var (
	weakPassOptions WeakPassOptions
)

func (o *WeakPassOptions) validateOptions() error {
	if weakPassOptions.Keywords == "" && !weakPassOptions.Show {
		return fmt.Errorf("please give keywords")
	}
	return nil
}
func init() {
	weakpassCmd := newCommandWithAliases("weakpass", "Start the application", []string{"weak", "wp", "wk", "ww"}, &weakPassOptions)
	rootCmd.AddCommand(weakpassCmd)

	weakpassCmd.PersistentFlags().StringVarP(&weakPassOptions.Keywords, "keyword", "k", "", "your keyword list, separate by ','")
	weakpassCmd.PersistentFlags().StringVarP(&weakPassOptions.Suffix, "suffix", "", strings.Join(common.SuffixTop, ","), "your suffix list, separate by ','")
	weakpassCmd.PersistentFlags().StringVarP(&weakPassOptions.Separator, "sep", "", "@,_", "your separator list, default: @,_ separate by ','")
	weakpassCmd.PersistentFlags().StringVarP(&weakPassOptions.Prefix, "prefix", "", "!,_", "your prefix list, default: null separate by ','")

	weakpassCmd.PersistentFlags().BoolVarP(&weakPassOptions.Full, "full", "", false, "full mode")
	weakpassCmd.PersistentFlags().BoolVarP(&weakPassOptions.ListFormat, "list", "l", false, "python list output")
	weakpassCmd.PersistentFlags().BoolVarP(&weakPassOptions.Variant, "variant", "", false, "if variant, eg: i -> 1")
	weakpassCmd.PersistentFlags().BoolVarP(&weakPassOptions.Show, "show", "", false, "show the entire list")

	viper.BindPFlag("keyword", weakpassCmd.PersistentFlags().Lookup("keyword"))
	viper.SetDefault("keyword", "")

	viper.BindPFlag("suffix", weakpassCmd.PersistentFlags().Lookup("suffix"))
	viper.SetDefault("suffix", strings.Join(common.SuffixTop, ","))

	viper.BindPFlag("sep", weakpassCmd.PersistentFlags().Lookup("sep"))
	viper.SetDefault("sep", "@,_")

	viper.BindPFlag("prefix", weakpassCmd.PersistentFlags().Lookup("prefix"))
	viper.SetDefault("prefix", "")

	viper.BindPFlag("full", weakpassCmd.PersistentFlags().Lookup("full"))
	viper.SetDefault("full", false)

	viper.BindPFlag("list", weakpassCmd.PersistentFlags().Lookup("list"))
	viper.SetDefault("list", false)

	viper.BindPFlag("variant", weakpassCmd.PersistentFlags().Lookup("variant"))
	viper.SetDefault("variant", false)
}

func (o *WeakPassOptions) run() {
	if weakPassOptions.Show {
		utils.ShowInfo()
		return
	}
	utils.GenerateWeakPassword()
}
