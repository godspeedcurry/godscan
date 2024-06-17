package cmd

import (
	"bufio"
	"fmt"
	"net"
	"net/url"
	"os"
	"strings"

	"github.com/godspeedcurry/godscan/utils"
	"github.com/spf13/viper"
)

func ContainPrivateIp(UrlList []string) bool {
	for _, Url := range UrlList {
		host, _ := url.Parse(Url)
		ip := net.ParseIP(host.Hostname())
		if ip != nil {
			return ip.IsPrivate()
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

	if ContainPrivateIp(targetUrlList) {
		reader := bufio.NewReader(os.Stdin)
		fmt.Printf("Do you want to scan private ip? (y/N) ")
		response, _ := reader.ReadString('\n')
		if strings.TrimSpace(response) == "y" || strings.TrimSpace(response) == "" {
			viper.Set("private-ip", true)
		}
	}
	return targetUrlList
}
