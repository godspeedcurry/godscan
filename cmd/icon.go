/*
Copyright © 2023 NAME HERE <EMAIL ADDRESS>
*/
package cmd

import (
	"fmt"

	"github.com/godspeedcurry/godscan/utils"
	"github.com/spf13/cobra"
)

type IconOptions struct {
}

var (
	iconOptions IconOptions
)

// iconCmd represents the icon command
var iconCmd = &cobra.Command{
	Use:   "icon",
	Short: "calculate hash of an icon, eg: godscan icon -u http://example.com/favicon.ico",
	Run: func(cmd *cobra.Command, args []string) {
		if err := iconOptions.validateOptions(); err != nil {
			fmt.Println("Try 'icon --url http://example.com/favicon.ico'")
			return
		}
		iconOptions.run()
	},
}

func init() {
	rootCmd.AddCommand(iconCmd)
}

func (o *IconOptions) validateOptions() error {
	if GlobalOption.Url == "" {
		return fmt.Errorf("please give target url")
	}
	return nil
}

func (o *IconOptions) run() {
	utils.IconDetect(GlobalOption.Url)
}
