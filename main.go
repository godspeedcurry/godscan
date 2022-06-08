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
}
