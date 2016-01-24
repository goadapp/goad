package main

import (
	"errors"
	"fmt"
	"github.com/gophergala2016/goad/Godeps/_workspace/src/github.com/aws/aws-sdk-go/aws"
	"github.com/gophergala2016/goad/queue"
	"io/ioutil"
	"net"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"
)

func main() {
	address := os.Args[1]
	concurrencycount, err := strconv.Atoi(os.Args[2])
	if err != nil {
		fmt.Printf("Error reading concurrency level: %s\n", err)
		return
	}
	maxRequestCount, err := strconv.Atoi(os.Args[3])
	sqsurl := os.Args[4]
	awsregion := os.Args[5]
	clientTimeout, _ := time.ParseDuration("1s")
	if len(os.Args) > 6 {
		newClientTimeout, err := time.ParseDuration(os.Args[6])
		if err == nil {
			clientTimeout = newClientTimeout
		} else {
			fmt.Printf("Error parsing timeout: %s\n", err)
			return
		}
	}
	fmt.Printf("Using a timeout of %s\n", clientTimeout)
	reportingFrequency, _ := time.ParseDuration("1s")
	if len(os.Args) > 7 {
		newReportingFrequency, err := time.ParseDuration(os.Args[7])
		if err == nil {
			reportingFrequency = newReportingFrequency
		} else {
			fmt.Printf("Error parsing reporting frequency: %s\n", err)
			return
		}
	}
	fmt.Printf("Using a reporting frequency of %s\n", reportingFrequency)

	client := &http.Client{}
	client.Timeout = clientTimeout
	client.CheckRedirect = func(req *http.Request, via []*http.Request) error {
		return errors.New("redirect")
	}
	fmt.Printf("Will spawn %d workers making %d requests to %s\n", concurrencycount, maxRequestCount, address)
	runLoadTest(client, sqsurl, address, maxRequestCount, concurrencycount, awsregion, reportingFrequency)
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

	awsConfig := aws.NewConfig().WithRegion(awsregion)
	sqsAdaptor := queue.NewSQSAdaptor(awsConfig, sqsurl)
	//sqsAdaptor := queue.NewDummyAdaptor(sqsurl)
func runLoadTest(client *http.Client, sqsurl string, url string, totalRequests int, concurrencycount int, awsregion string, reportingFrequency time.Duration) {
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
		go fetch(loadTestStartTime, client, url, totalRequests, jobs, ch, &wg, awsregion)
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
					float32(i) / durationSeconds,
					(float32(totBytesRead) / durationSeconds) / 1024.0,
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

func fetch(loadTestStartTime time.Time, client *http.Client, address string, requestcount int, jobs <-chan struct{}, ch chan RequestResult, wg *sync.WaitGroup, awsregion string) {
	defer wg.Done()
	for _ = range jobs {
		start := time.Now()
		req, err := http.NewRequest("GET", address, nil)
		req.Header.Add("User-Agent", "Mozilla/5.0 (compatible; Goad/1.0; +https://goad.io)")
		req.Header.Add("Accept-Encoding", "gzip")
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
		if err != nil && !strings.Contains(err.Error(), "redirect") {
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
			_, err = response.Body.Read(buf)
			firstByteRead := true
			if err != nil {
				status = fmt.Sprintf("reading first byte failed: %s\n", err)
				firstByteRead = false
			}
			elapsedFirstByte = time.Since(start)
			body, err := ioutil.ReadAll(response.Body)
			response.Body.Close()
			if firstByteRead {
				bytesRead = len(body) + 1
			}
			elapsedLastByte = time.Since(start)
			if err != nil {
				status = fmt.Sprintf("reading response body failed: %s\n", err)
			} else {
				status = "Success"
			}
			elapsed = time.Since(start)
		}
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
