package main

import (
	"flag"
	"fmt"
	"os"
	"os/signal"
	"sort"
	"strconv"
	"syscall"
	"time"

	"github.com/gophergala2016/goad"
	"github.com/gophergala2016/goad/Godeps/_workspace/src/github.com/dustin/go-humanize"
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
	flag.StringVar(&region, "r", "us-east-1", "AWS regions to run in")
	flag.Parse()

	if len(flag.Args()) < 1 {
		fmt.Println("You must specify a URL as a last argument")
		os.Exit(1)
	}

	url = flag.Args()[0]

	test, testerr := goad.NewTest(&goad.TestConfig{
		URL:            url,
		Concurrency:    concurrency,
		TotalRequests:  requests,
		RequestTimeout: time.Duration(timeout) * time.Second,
		Region:         region,
	})
	if testerr != nil {
		fmt.Println(testerr)
		os.Exit(1)
	}

	var finalResult queue.RegionsAggData
	defer printSummary(&finalResult)

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM) // but interrupts from kbd are blocked by termbox

	start(test, &finalResult, sigChan)
}

func start(test *goad.Test, finalResult *queue.RegionsAggData, sigChan chan os.Signal) {
	err := termbox.Init()
	if err != nil {
		panic(err)
	}

	defer termbox.Close()

	resultChan := test.Start()
	termbox.Sync()
	renderString(0, 0, "Launching on AWS...", coldef, coldef)
	_, h := termbox.Size()
	renderString(0, h-1, "Press ctrl-c to interrupt", coldef, coldef)
	termbox.Flush()

	go func() {
		for {
			event := termbox.PollEvent()
			if event.Key == 3 {
				sigChan <- syscall.SIGINT
			}
		}
	}()

outer:
	for {
		select {
		case result, ok := <-resultChan:
			if !ok {
				break outer
			}
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

		case <-sigChan:
			break outer
		}
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
	headingStr := "   TotReqs   TotBytes    AvgTime   AvgReq/s"
	renderString(x, y, headingStr, coldef|termbox.AttrBold, coldef)
	y++
	resultStr := fmt.Sprintf("%10d %10s   %7.3fs %10.2f", data.TotalReqs, humanize.Bytes(uint64(data.TotBytesRead)), float64(data.AveTimeForReq)/nano, data.AveReqPerSec)
	renderString(x, y, resultStr, coldef, coldef)
	y++
	headingStr = "   Slowest    Fastest   Timeouts  TotErrors"
	renderString(x, y, headingStr, coldef|termbox.AttrBold, coldef)
	y++
	resultStr = fmt.Sprintf("  %7.3fs   %7.3fs %10d %10d", float64(data.Slowest)/nano, float64(data.Fastest)/nano, data.TotalTimedOut, totErrors(&data))
	renderString(x, y, resultStr, coldef, coldef)
	y++

	return y
}

func totErrors(data *queue.AggData) int {
	var okReqs int
	for statusStr, value := range data.Statuses {
		status, _ := strconv.Atoi(statusStr)
		if status >= 200 && status <= 299 {
			okReqs += value
		}
	}
	return data.TotalReqs - okReqs
}

func drawProgressBar(percent float64, y int) {
	x := 0
	percentStr := fmt.Sprintf("%5.1f%%            ", percent*100)
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

func printData(data *queue.AggData) {
	boldPrintln("   TotReqs   TotBytes    AvgTime   AvgReq/s")
	fmt.Printf("%10d %10s   %7.3fs %10.2f\n", data.TotalReqs, humanize.Bytes(uint64(data.TotBytesRead)), float64(data.AveTimeForReq)/nano, data.AveReqPerSec)
	boldPrintln("   Slowest    Fastest   Timeouts  TotErrors")
	fmt.Printf("  %7.3fs   %7.3fs %10d %10d", float64(data.Slowest)/nano, float64(data.Fastest)/nano, data.TotalTimedOut, totErrors(data))
	fmt.Println("")
}

func printSummary(result *queue.RegionsAggData) {
	if len(result.Regions) == 0 {
		boldPrintln("No results received")
		return
	}
	boldPrintln("Regional results")
	fmt.Println("")

	for region, data := range result.Regions {
		fmt.Println("Region: " + region)
		printData(&data)
	}

	overall := queue.SumRegionResults(result)

	fmt.Println("")
	boldPrintln("Overall")
	fmt.Println("")
	printData(overall)

	boldPrintln("HTTPStatus   Requests")
	for statusStr, value := range overall.Statuses {
		fmt.Printf("%10s %10d\n", statusStr, value)
	}
	fmt.Println("")
}
