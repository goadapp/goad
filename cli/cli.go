package main

import (
	"flag"
	"fmt"
	"os"
	"time"

	"github.com/gophergala2016/goad"
	"github.com/nsf/termbox-go"
)

var (
	url         string
	concurrency uint
	requests    uint
	timeout     uint
	region      string
)

const coldef = termbox.ColorDefault
const nano = 1000000000

func main() {
	termbox.Init()
	defer termbox.Close()

	flag.UintVar(&concurrency, "c", 10, "number of concurrent requests")
	flag.UintVar(&requests, "n", 1000, "number of total requests to make")
	flag.UintVar(&timeout, "t", 15, "request timeout in seconds")
	flag.StringVar(&region, "r", "us-east-1", "AWS region")
	flag.Parse()

	if len(flag.Args()) < 1 {
		fmt.Println("You must specify a URL")
		os.Exit(1)
	}

	url = flag.Args()[0]

	test := goad.NewTest(&goad.TestConfig{
		URL:            url,
		Concurrency:    concurrency,
		TotalRequests:  requests,
		RequestTimeout: time.Duration(timeout) * time.Second,
		Region:         region,
	})

	resultChan := test.Start()
	termbox.Sync()
	for result := range resultChan {
		//		fmt.Printf("%#v\n", result)
		x := 0
		y := 0
		// must sort these somehow
		for key, value := range result.Regions {
			regionStr := fmt.Sprintf("%-10s", key)
			for i, c := range regionStr {
				termbox.SetCell(x+i, y, c, coldef, coldef)
			}
			y++
			headingStr := "   TotReqs   TotBytes  AveTime    AveReq/s Ave1stByte"
			for i, c := range headingStr {
				termbox.SetCell(x+i, y, c, coldef, coldef)
			}
			y++
			resultStr := fmt.Sprintf("%10d %10d %7.2fs %10.2f   %7.2fs", value.TotalReqs, value.TotBytesRead, float64(value.AveTimeForReq)/nano, value.AveReqPerSec, float64(value.AveTimeToFirst)/nano)
			x = 0
			for i, c := range resultStr {
				termbox.SetCell(x+i, y, c, coldef, coldef)
			}
			termbox.Flush()
		}
	}
}
