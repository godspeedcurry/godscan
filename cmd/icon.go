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
	iconURL, err := utils.FindFaviconURL(GlobalOption.Url)
	if err != nil {
		utils.Error("%v", err)
		return
	}
	fofa, hunter, iconB64, err := utils.IconDetect(iconURL)
	if err != nil {
		utils.Error("%v", err)
		return
	}
	utils.Info("icon_url: %s\nfofa: icon_hash=\"%s\"\nhunter: web.icon=\"%s\"\nbase64 (len=%d): %s\n", iconURL, fofa, hunter, len(iconB64), iconB64)
}
