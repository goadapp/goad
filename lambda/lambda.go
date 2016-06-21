package main

import (
	"bytes"
	"crypto/tls"
	"flag"
	"fmt"
	"io/ioutil"
	"net"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/goadapp/goad/helpers"
	"github.com/goadapp/goad/queue"
)

func main() {

	var (
		address          string
		sqsurl           string
		concurrencycount int
		maxRequestCount  int
		timeout          string
		frequency        string
		awsregion        string
		queueRegion      string
		requestMethod    string
		requestBody      string
		hostHeader       string
		requestHeaders   helpers.StringsliceFlag
	)

	flag.StringVar(&address, "u", "", "URL to load test (required)")
	flag.StringVar(&requestMethod, "m", "GET", "HTTP method")
	flag.StringVar(&requestBody, "b", "", "HTTP request body")
	flag.StringVar(&awsregion, "r", "", "AWS region to run in")
	flag.StringVar(&queueRegion, "q", "", "Queue region")
	flag.StringVar(&sqsurl, "s", "", "sqsUrl")
	flag.StringVar(&timeout, "t", "15s", "request timeout in seconds")
	flag.StringVar(&frequency, "f", "15s", "Reporting frequency in seconds")
	flag.StringVar(&hostHeader, "e", "", "explicit host header to send")

	flag.IntVar(&concurrencycount, "c", 10, "number of concurrent requests")
	flag.IntVar(&maxRequestCount, "n", 1000, "number of total requests to make")

	flag.Var(&requestHeaders, "H", "List of headers")
	flag.Parse()

	clientTimeout, _ := time.ParseDuration(timeout)
	fmt.Printf("Using a timeout of %s\n", clientTimeout)
	reportingFrequency, _ := time.ParseDuration(frequency)
	fmt.Printf("Using a reporting frequency of %s\n", reportingFrequency)

	// InsecureSkipVerify so that sites with self signed certs can be tested
	tr := &http.Transport{
		TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
	}
	client := &http.Client{Transport: tr}
	client.Timeout = clientTimeout

	fmt.Printf("Will spawn %d workers making %d requests to %s\n", concurrencycount, maxRequestCount, address)
	runLoadTest(client, sqsurl, address, maxRequestCount, concurrencycount, awsregion, reportingFrequency, queueRegion, requestMethod, requestBody, hostHeader, requestHeaders)
}

type RequestResult struct {
	Time             int64  `json:"time"`
	Host             string `json:"host"`
	Type             string `json:"type"`
	Status           int    `json:"status"`
	ElapsedFirstByte int64  `json:"elapsed-first-byte"`
	ElapsedLastByte  int64  `json:"elapsed-last-byte"`
	Elapsed          int64  `json:"elapsed"`
	Bytes            int    `json:"bytes"`
	Timeout          bool   `json:"timeout"`
	ConnectionError  bool   `json:"connection-error"`
	State            string `json:"state"`
}

func runLoadTest(client *http.Client, sqsurl string, url string, totalRequests int, concurrencycount int, awsregion string, reportingFrequency time.Duration, queueRegion string, requestMethod string, requestBody string, hostHeader string, requestHeaders []string) {
	awsConfig := aws.NewConfig().WithRegion(queueRegion)
	sqsAdaptor := queue.NewSQSAdaptor(awsConfig, sqsurl)
	//sqsAdaptor := queue.NewDummyAdaptor(sqsurl)
	jobs := make(chan struct{}, totalRequests)
	ch := make(chan RequestResult, totalRequests)
	var wg sync.WaitGroup
	loadTestStartTime := time.Now()
	var requestsSoFar int
	for i := 0; i < totalRequests; i++ {
		jobs <- struct{}{}
	}
	close(jobs)
	fmt.Print("Spawning workers…")
	for i := 0; i < concurrencycount; i++ {
		wg.Add(1)
		go fetch(loadTestStartTime, client, url, totalRequests, jobs, ch, &wg, awsregion, requestMethod, requestBody, hostHeader, requestHeaders)
		fmt.Print(".")
	}
	fmt.Println(" done.\nWaiting for results…")

	ticker := time.NewTicker(reportingFrequency)
	quit := make(chan struct{})
	quitting := false

	for requestsSoFar < totalRequests && !quitting {
		i := 0

		var timeToFirstTotal int64
		var requestTimeTotal int64
		totBytesRead := 0
		statuses := make(map[string]int)
		var firstRequestTime int64
		var lastRequestTime int64
		var slowest int64
		var fastest int64
		var totalTimedOut int
		var totalConnectionError int

		resetStats := false
		for requestsSoFar < totalRequests && !quitting && !resetStats {
			aggregate := false
			select {
			case r := <-ch:
				i++
				requestsSoFar++
				if requestsSoFar%10 == 0 || requestsSoFar == totalRequests {
					fmt.Printf("\r%.2f%% done (%d requests out of %d)", (float64(requestsSoFar)/float64(totalRequests))*100.0, requestsSoFar, totalRequests)
				}
				if firstRequestTime == 0 {
					firstRequestTime = r.Time
				}

				lastRequestTime = r.Time

				if r.Timeout {
					totalTimedOut++
					continue
				}
				if r.ConnectionError {
					totalConnectionError++
					continue
				}

				if r.ElapsedLastByte > slowest {
					slowest = r.ElapsedLastByte
				}
				if fastest == 0 {
					fastest = r.ElapsedLastByte
				} else {
					if r.ElapsedLastByte < fastest {
						fastest = r.ElapsedLastByte
					}
				}

				timeToFirstTotal += r.ElapsedFirstByte
				totBytesRead += r.Bytes
				statusStr := strconv.Itoa(r.Status)
				_, ok := statuses[statusStr]
				if !ok {
					statuses[statusStr] = 1
				} else {
					statuses[statusStr]++
				}
				requestTimeTotal += r.Elapsed
				if requestsSoFar == totalRequests {
					quitting = true
				}
			case <-ticker.C:
				if i == 0 {
					continue
				}
				aggregate = true
			case <-quit:
				ticker.Stop()
				quitting = true
			}
			if aggregate || quitting {
				durationNanoSeconds := lastRequestTime - firstRequestTime
				durationSeconds := float32(durationNanoSeconds) / float32(1000000000)
				var reqPerSec float32
				var kbPerSec float32
				if durationSeconds > 0 {
					reqPerSec = float32(i) / durationSeconds
					kbPerSec = (float32(totBytesRead) / durationSeconds) / 1024.0
				} else {
					reqPerSec = 0
					kbPerSec = 0
				}

				fatalError := ""
				if (totalTimedOut + totalConnectionError) > i/2 {
					fatalError = "Over 50% of requests failed, aborting"
					quitting = true
				}
				aggData := queue.AggData{
					i,
					totalTimedOut,
					totalConnectionError,
					timeToFirstTotal / int64(i),
					totBytesRead,
					statuses,
					requestTimeTotal / int64(i),
					reqPerSec,
					kbPerSec,
					slowest,
					fastest,
					awsregion,
					fatalError,
				}
				sqsAdaptor.SendResult(aggData)
				resetStats = true
			}
		}
	}
	fmt.Printf("\nYay🎈  - %d requests completed\n", requestsSoFar)

}

func fetch(loadTestStartTime time.Time, client *http.Client, address string, requestcount int, jobs <-chan struct{}, ch chan RequestResult, wg *sync.WaitGroup, awsregion string, requestMethod string, requestBody string, hostHeader string, requestHeaders []string) {
	defer wg.Done()
	for _ = range jobs {
		start := time.Now()
		req, err := http.NewRequest(requestMethod, address, bytes.NewBufferString(requestBody))
		if err != nil {
			fmt.Println("Error creating the HTTP request:", err)
			return
		}
		req.Header.Add("User-Agent", "Mozilla/5.0 (compatible; Goad/1.0; +https://goad.io)")
		req.Header.Add("Accept-Encoding", "gzip")
		if hostHeader != "" {
			req.Host = strings.Trim(hostHeader, " ")
		}
		for _, v := range requestHeaders {
			header := strings.Split(v, ":")
			if strings.ToLower(strings.Trim(header[0], " ")) == "host" {
				req.Host = strings.Trim(header[1], " ")
			} else {
				req.Header.Add(strings.Trim(header[0], " "), strings.Trim(header[1], " "))
			}
		}

		response, err := client.Do(req)
		var status string
		var elapsedFirstByte time.Duration
		var elapsedLastByte time.Duration
		var elapsed time.Duration
		var statusCode int
		var bytesRead int
		buf := []byte(" ")
		timedOut := false
		connectionError := false
		isRedirect := err != nil && strings.Contains(err.Error(), "redirect")
		if err != nil && !isRedirect {
			status = fmt.Sprintf("ERROR: %s\n", err)
			switch err := err.(type) {
			case *url.Error:
				if err, ok := err.Err.(net.Error); ok && err.Timeout() {
					timedOut = true
				}
			case net.Error:
				if err.Timeout() {
					timedOut = true
				}
			}

			if !timedOut {
				connectionError = true
			}
		} else {
			statusCode = response.StatusCode
			elapsedFirstByte = time.Since(start)
			if !isRedirect {
				_, err = response.Body.Read(buf)
				firstByteRead := true
				if err != nil {
					status = fmt.Sprintf("reading first byte failed: %s\n", err)
					firstByteRead = false
				}
				body, err := ioutil.ReadAll(response.Body)
				if firstByteRead {
					bytesRead = len(body) + 1
				}
				elapsedLastByte = time.Since(start)
				if err != nil {
					// todo: detect timeout here as well
					status = fmt.Sprintf("reading response body failed: %s\n", err)
					connectionError = true
				} else {
					status = "Success"
				}
			} else {
				status = "Redirect"
			}
			response.Body.Close()

			elapsed = time.Since(start)
		}
		//fmt.Printf("Request end: %d, elapsed: %d\n", time.Now().Sub(loadTestStartTime).Nanoseconds(), elapsed.Nanoseconds())
		result := RequestResult{
			start.Sub(loadTestStartTime).Nanoseconds(),
			req.URL.Host,
			req.Method,
			statusCode,
			elapsed.Nanoseconds(),
			elapsedFirstByte.Nanoseconds(),
			elapsedLastByte.Nanoseconds(),
			bytesRead,
			timedOut,
			connectionError,
			status,
		}
		ch <- result
	}
}
