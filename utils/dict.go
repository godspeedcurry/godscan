package utils

import (
	"fmt"
	"github.com/gobuffalo/packr"
	"strings"
)

var box = packr.NewBox("../dict")

func Get_web_username() []string {
	var web_username = box.String("web/web_username.txt")
	var web_username_arr = strings.Split(web_username, "\n")
	return web_username_arr
}

func Get_web_password() []string {
	var web_password = box.String("web/web_password.txt")
	var web_password_arr = strings.Split(web_password, "\n")
	return web_password_arr
}

func Get_protocol_username(protocol string) []string {
	var protocol_username = box.String(fmt.Sprintf("user/dic_username_%s.txt", protocol))
	var protocol_username_arr = strings.Split(protocol_username, "\n")
	return protocol_username_arr
}

func Get_protocol_password(protocol string) []string {
	var protocol_password = box.String(fmt.Sprintf("pass/dic_password_%s.txt", protocol))
	var protocol_password_arr = strings.Split(protocol_password, "\n")
	return protocol_password_arr
}

func Get_device_username(device string) []string {
	var device_username = box.String(fmt.Sprintf("device/%s/device_username_%s.txt", device, device))
	var device_username_arr = strings.Split(device_username, "\n")
	return device_username_arr
}

func Get_device_password(device string) []string {
	var device_password = box.String(fmt.Sprintf("device/%s/device_password_%s.txt", device, device))
	var device_password_arr = strings.Split(device_password, "\n")
	return device_password_arr
}
