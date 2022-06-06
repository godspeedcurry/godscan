package main

import (
	"main/utils"
	// "github.com/gobuffalo/packr"
	"flag"
)

var (
	Url   string
	Proxy string
	Depth int
)

func init() {
	flag.StringVar(&Url, "u", "https://www.baidu.com", "your url")
	flag.StringVar(&Proxy, "p", "", "your proxy")
	flag.IntVar(&Depth, "d", 1, "your search depth")
}
func main() {
	flag.Parse()
	utils.PrintFinger(Url)

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
