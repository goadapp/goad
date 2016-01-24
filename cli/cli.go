package main

import (
	"flag"
	"fmt"
	"os"
	"sort"
	"time"

	"github.com/gophergala2016/goad"
	"github.com/gophergala2016/goad/Godeps/_workspace/src/github.com/nsf/termbox-go"
	"github.com/gophergala2016/goad/queue"
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

	var finalResult queue.RegionsAggData
	defer printSummary(&finalResult)

	start(test, &finalResult)
}

func start(test *goad.Test, finalResult *queue.RegionsAggData) {
	err := termbox.Init()
	if err != nil {
		panic(err)
	}
	defer termbox.Close()

	resultChan := test.Start()
	termbox.Sync()
	for result := range resultChan {
		//		result.Regions["eu-west-1"] = result.Regions["us-east-1"]
		// sort so that regions always appear in the same order
		var regions []string
		for key := range result.Regions {
			regions = append(regions, key)
		}
		sort.Strings(regions)
		y := 3
		totalReqs := 0
		for _, region := range regions {
			data := result.Regions[region]
			totalReqs += data.TotalReqs
			y = renderRegion(data, y)
			y++
		}

		y = 0
		percentDone := float64(totalReqs) / float64(result.TotalExpectedRequests)
		drawProgressBar(percentDone, y)

		termbox.Flush()
		finalResult.Regions = result.Regions
	}
}

// renderRegion returns the y for the next empty line
func renderRegion(data queue.AggData, y int) int {
	x := 0
	renderString(x, y, "Region: ", termbox.ColorWhite, termbox.ColorBlue)
	x += 8
	regionStr := fmt.Sprintf("%-10s", data.Region)
	renderString(x, y, regionStr, termbox.ColorWhite|termbox.AttrBold, termbox.ColorBlue)
	x = 0
	y++
	headingStr := "   TotReqs   TotBytes  AveTime   AveReq/s Ave1stByte"
	renderString(x, y, headingStr, coldef|termbox.AttrBold, coldef)
	y++
	resultStr := fmt.Sprintf("%10d %10d %7.2fs %10.2f   %7.2fs", data.TotalReqs, data.TotBytesRead, float64(data.AveTimeForReq)/nano, data.AveReqPerSec, float64(data.AveTimeToFirst)/nano)
	x = 0
	renderString(x, y, resultStr, coldef, coldef)
	y++
	return y
}

func drawProgressBar(percent float64, y int) {
	x := 0
	percentStr := fmt.Sprintf("%5.1f%%", percent*100)
	renderString(x, y, percentStr, coldef, coldef)
	y++

	hashes := int(percent * 50)
	if percent > 0.98 {
		hashes = 50
	}
	renderString(x, y, "[", coldef, coldef)

	for x++; x <= hashes; x++ {
		renderString(x, y, "#", coldef, coldef)
	}
	renderString(51, y, "]", coldef, coldef)
}

func renderString(x int, y int, str string, f termbox.Attribute, b termbox.Attribute) {
	for i, c := range str {
		termbox.SetCell(x+i, y, c, f, b)
	}
}

func boldPrintln(msg string) {
	fmt.Printf("\033[1m%s\033[0m\n", msg)
}

func printSummary(result *queue.RegionsAggData) {
	boldPrintln("Summary")
	fmt.Println("")
	for region, data := range result.Regions {
		fmt.Println("Region: " + region)
		boldPrintln("   TotReqs   TotBytes  AveTime   AveReq/s Ave1stByte")
		fmt.Printf("%10d %10d %7.2fs %10.2f   %7.2fs\n", data.TotalReqs, data.TotBytesRead, float64(data.AveTimeForReq)/nano, data.AveReqPerSec, float64(data.AveTimeToFirst)/nano)
		fmt.Println("")
	}
}
