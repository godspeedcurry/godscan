package cmd

import (
	"bufio"
	"net"
	"net/url"
	"os"
	"strings"

	"github.com/godspeedcurry/godscan/utils"
	"github.com/spf13/viper"
)

func ContainPrivateIp(UrlList []string) bool {
	for _, Url := range UrlList {
		host, err := url.Parse(Url)
		if err != nil {
			utils.Error("Failed to parse URL %s: %v", Url, err)
			continue
		}
		ip := net.ParseIP(host.Hostname())
		if ip != nil {
			if ip.IsPrivate() {
				return true
			}
			continue
		}
	}
	return false
}

func GetTargetList() []string {
	targetUrlList := []string{}

	if GlobalOption.Url != "" {
		targetUrlList = append(targetUrlList, GlobalOption.Url)
	} else {
		targetUrlList = append(targetUrlList, utils.FilReadUrl(GlobalOption.UrlFile)...)
	}

	if ContainPrivateIp(targetUrlList) && !viper.GetBool("private-ip") {
		reader := bufio.NewReader(os.Stdin)
		utils.Info("Do you want to scan private ip? (y/N)")
		response, err := reader.ReadString('\n')
		if err != nil {
			utils.Error("Failed to read input: %v", err)
			return targetUrlList
		}
		if strings.TrimSpace(response) == "y" || strings.TrimSpace(response) == "" {
			viper.Set("private-ip", true)
		}
	}
	return targetUrlList
}
