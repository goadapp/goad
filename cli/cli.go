package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"io/ioutil"
	"os"
	"os/signal"
	"sort"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/dustin/go-humanize"
	"github.com/goadapp/goad"
	"github.com/goadapp/goad/helpers"
	"github.com/goadapp/goad/queue"
	"github.com/goadapp/goad/version"
	"github.com/nsf/termbox-go"
)

var (
	url         string
	concurrency uint
	requests    uint
	timeout     uint
	regions     string
	method      string
	body        string
	headers     helpers.StringsliceFlag
	awsProfile  string
	outputFile  string
)

const coldef = termbox.ColorDefault
const nano = 1000000000
const messageRows = 8
const messageStartRow = 9

func main() {
	var printVersion bool

	flag.StringVar(&url, "u", "", "URL to load test (required)")
	flag.StringVar(&method, "m", "GET", "HTTP method")
	flag.StringVar(&body, "b", "", "HTTP request body")
	flag.UintVar(&concurrency, "c", 10, "number of concurrent requests")
	flag.UintVar(&requests, "n", 1000, "number of total requests to make")
	flag.UintVar(&timeout, "t", 15, "request timeout in seconds")
	flag.StringVar(&regions, "r", "us-east-1,eu-west-1,ap-northeast-1", "AWS regions to run in (comma separated, no spaces)")
	flag.StringVar(&awsProfile, "p", "", "AWS named profile to use")
	flag.StringVar(&outputFile, "o", "", "Optional path to JSON file for result storage")
	flag.Var(&headers, "H", "List of headers")
	flag.BoolVar(&printVersion, "version", false, "print the current Goad version")
	flag.Parse()

	if printVersion {
		fmt.Println(version.Version)
		os.Exit(0)
	}

	if url == "" {
		flag.Usage()
		os.Exit(0)
	}

	test, testerr := goad.NewTest(&goad.TestConfig{
		URL:            url,
		Concurrency:    concurrency,
		TotalRequests:  requests,
		RequestTimeout: time.Duration(timeout) * time.Second,
		Regions:        strings.Split(regions, ","),
		Method:         method,
		Body:           body,
		Headers:        headers,
		AwsProfile:     awsProfile,
	})
	if testerr != nil {
		fmt.Println(testerr)
		os.Exit(1)
	}

	var finalResult queue.RegionsAggData
	defer printSummary(&finalResult)

	if outputFile != "" {
		defer saveJSONSummary(outputFile, &finalResult)
	}

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
	termbox.Sync()
	renderLogo()
	termbox.Flush()

	messages := make(chan string)
	go receiveMessages(messages)
	messages <- "Launching on AWS..."
	resultChan := test.Start(messages)

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

	firstTime := true
outer:
	for {
		select {
		case result, ok := <-resultChan:
			if !ok {
				break outer
			}

			if firstTime {
				clearLogo()
				clearMessages()
				firstTime = false
			}

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

func receiveMessages(messages chan string) {
	curRow := messageStartRow
	totReceived := 0
	messagesReceived := []string{}
	for {
		msg := <-messages
		if curRow == messageStartRow+messageRows-1 {
			// scroll
			for i := 0; i < messageRows-1; i++ {
				renderString(0, i+messageStartRow, fmt.Sprintf("%40s", ""), coldef, coldef)
				renderString(0, i+messageStartRow, messagesReceived[totReceived-messageRows+1+i], coldef, coldef)
			}
			renderString(0, curRow, fmt.Sprintf("%40s", ""), coldef, coldef)
		}
		messagesReceived = append(messagesReceived, msg)
		totReceived++
		renderString(0, curRow, msg, coldef, coldef)
		termbox.Flush()
		if curRow < messageStartRow+messageRows-1 {
			curRow++
		}
	}
}

func renderLogo() {
	s1 := `	  _____                 _`
	s2 := `  / ____|               | |`
	s3 := `	| |  __  ___   ____  __| |`
	s4 := `	| | |_ |/ _ \ / _  |/ _  |`
	s5 := `	| |__| | (_) | (_| | (_| |`
	s6 := `	 \_____|\___/ \__,_|\__,_|`
	s7 := " Global load testing with Go"
	arr := [...]string{s1, s2, s3, s4, s5, s6, s7}
	for i, str := range arr {
		renderString(0, i, str, coldef, coldef)
	}
}

func clearLogo() {
	for i := 0; i < 7; i++ {
		renderString(0, i, fmt.Sprintf("%32s", ""), coldef, coldef)
	}
}

func clearMessages() {
	for i := 0; i < messageRows; i++ {
		renderString(0, i+messageStartRow, fmt.Sprintf("%40s", ""), coldef, coldef)
	}
}

// renderRegion returns the y for the next empty line
func renderRegion(data queue.AggData, y int) int {
	x := 0
	renderString(x, y, "Region: ", termbox.ColorWhite, termbox.ColorBlue)
	x += 8
	regionStr := fmt.Sprintf("%s", data.Region)
	renderString(x, y, regionStr, termbox.ColorWhite|termbox.AttrBold, termbox.ColorBlue)
	x = 0
	y++
	headingStr := "   TotReqs   TotBytes    AvgTime   AvgReq/s  AvgKbps/s"
	renderString(x, y, headingStr, coldef|termbox.AttrBold, coldef)
	y++
	resultStr := fmt.Sprintf("%10d %10s   %7.3fs %10.2f %10.2f", data.TotalReqs, humanize.Bytes(uint64(data.TotBytesRead)), float64(data.AveTimeForReq)/nano, data.AveReqPerSec, data.AveKBytesPerSec)
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
		if status < 400 {
			okReqs += value
		}
	}
	return data.TotalReqs - okReqs
}

func drawProgressBar(percent float64, y int) {
	x := 0
	width := 52
	percentStr := fmt.Sprintf("%5.1f%%            ", percent*100)
	renderString(x, y, percentStr, coldef, coldef)
	y++
	hashes := int(percent * float64(width))
	if percent > 0.99 {
		hashes = width
	}
	renderString(x, y, "[", coldef, coldef)

	for x++; x <= hashes; x++ {
		renderString(x, y, "#", coldef, coldef)
	}
	renderString(width+1, y, "]", coldef, coldef)
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
	boldPrintln("   TotReqs   TotBytes    AvgTime   AvgReq/s  AvgKbps/s")
	fmt.Printf("%10d %10s   %7.3fs %10.2f %10.2f\n", data.TotalReqs, humanize.Bytes(uint64(data.TotBytesRead)), float64(data.AveTimeForReq)/nano, data.AveReqPerSec, data.AveKBytesPerSec)
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

func saveJSONSummary(path string, result *queue.RegionsAggData) {
	if len(result.Regions) == 0 {
		return
	}
	results := make(map[string]queue.AggData)

	for region, data := range result.Regions {
		results[region] = data
	}

	overall := queue.SumRegionResults(result)

	results["overall"] = *overall
	b, err := json.MarshalIndent(results, "", "  ")
	if err != nil {
		fmt.Println(err)
		return
	}
	err = ioutil.WriteFile(path, b, 0644)
	if err != nil {
		fmt.Println(err)
		return
	}
}
