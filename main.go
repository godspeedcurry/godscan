package main

import (
	"main/utils"
	// "github.com/gobuffalo/packr"
	"flag"
)

var (
	url   string
	proxy string
)

func init() {
	flag.StringVar(&url, "u", "https://www.baidu.com", "your url")
	flag.StringVar(&proxy, "p", "", "your proxy")
}
func main() {
	flag.Parse()
	utils.PrintFinger(url)
	// var web_username = dict.Get_web_username()
	// var device_username = dict.Get_device_password("topsec")
	// color.Cyan("Prints text in cyan.")
	// // box := packr.NewBox("./dict")
	// // data := box.String("web/web_username.txt")
	// // var slices = strings.Split(data, "\n")
	// // fmt.Println(web_username)
	// fmt.Printf(fmt.Sprintf("adsa%s","123"))
	// for index,value := range device_username{
	// 	fmt.Printf("line%d=%s\n", index,value)
	// }
}
