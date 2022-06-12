package main

import (
	"flag"
	"main/utils"
)

var (
	Url      string
	Proxy    string
	Depth    int
	Keywords string
)

func init() {
	flag.StringVar(&Url, "u", "", "your url")
	flag.StringVar(&Proxy, "p", "", "your proxy")
	flag.IntVar(&Depth, "d", 1, "your search depth")
	flag.StringVar(&Keywords, "k", "", "your keyword list, separate by `,`")
}

func main() {
	flag.Parse()
	if Url != "" {
		utils.PrintFinger(Url)
	}
	if Keywords != "" {
		utils.GeneateWeakPassword(Keywords)
	}
}
