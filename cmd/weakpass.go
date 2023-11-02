/*
Copyright © 2023 NAME HERE <EMAIL ADDRESS>
*/
package cmd

import (
	"fmt"

	"github.com/godspeedcurry/godscan/utils"
	"github.com/spf13/cobra"
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

var weakpassCmd = &cobra.Command{
	Use:   "weakpass",
	Short: "generate weakpass word",
	Run: func(cmd *cobra.Command, args []string) {
		if err := weakPassOptions.validateOptions(); err != nil {
			fmt.Println("Try 'weakpass -k \"baidu\"'")
			return
		}
		weakPassOptions.run()
	},
}

func (o *WeakPassOptions) validateOptions() error {
	if weakPassOptions.Keywords == "" && weakPassOptions.Show == false {
		return fmt.Errorf("please give keywords")
	}
	return nil
}
func init() {
	rootCmd.AddCommand(weakpassCmd)

	weakpassCmd.PersistentFlags().StringVarP(&weakPassOptions.Keywords, "keyword", "k", "", "your keyword list, separate by ','")
	weakpassCmd.PersistentFlags().StringVarP(&weakPassOptions.Suffix, "suffix", "", "123,WSX,888,01,1,#", "your suffix list, default: 123,WSX,888 separate by ','")
	weakpassCmd.PersistentFlags().StringVarP(&weakPassOptions.Separator, "sep", "", "@,#,$", "your separator list, default: @,#,$ separate by ','")
	weakpassCmd.PersistentFlags().StringVarP(&weakPassOptions.Prefix, "prefix", "", "", "your prefix list, default: null separate by ','")

	weakpassCmd.PersistentFlags().BoolVarP(&weakPassOptions.Full, "full", "", false, "full mode")
	weakpassCmd.PersistentFlags().BoolVarP(&weakPassOptions.ListFormat, "list", "l", false, "python list output")
	weakpassCmd.PersistentFlags().BoolVarP(&weakPassOptions.Variant, "variant", "", false, "if variant, eg: i -> 1")
	weakpassCmd.PersistentFlags().BoolVarP(&weakPassOptions.Show, "show", "", false, "if show default args")

	viper.BindPFlag("keyword", weakpassCmd.PersistentFlags().Lookup("keyword"))
	viper.SetDefault("keyword", "")

	viper.BindPFlag("suffix", weakpassCmd.PersistentFlags().Lookup("suffix"))
	viper.SetDefault("suffix", "123,WSX,888,01,1,#")

	viper.BindPFlag("sep", weakpassCmd.PersistentFlags().Lookup("sep"))
	viper.SetDefault("sep", "@,#,$")

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
