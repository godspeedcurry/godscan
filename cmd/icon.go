/*
Copyright © 2023 NAME HERE <EMAIL ADDRESS>
*/
package cmd

import (
	"fmt"

	"github.com/godspeedcurry/godscan/utils"
)

type IconOptions struct {
}

var (
	iconOptions IconOptions
)

// iconCmd represents the icon command

func init() {
	iconCmd := newCommandWithAliases("icon", "Calculate hash of an icon, eg: godscan icon -u http://example.com/favicon.ico", []string{"ico"}, &iconOptions)
	rootCmd.AddCommand(iconCmd)
}

func (o *IconOptions) validateOptions() error {
	if GlobalOption.Url == "" {
		return fmt.Errorf("please give target url")
	}
	return nil
}

func (o *IconOptions) run() {
	utils.InitHttp()
	utils.IconDetect(GlobalOption.Url)
}
