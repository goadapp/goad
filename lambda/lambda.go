package main

import (
	"fmt"
	"github.com/gophergala2016/goad/sqsadaptor"
	"io/ioutil"
	"net/http"
	"os"
	"strconv"
	"sync"
	"time"
)

func main() {
	address := os.Args[1]
	concurrencycount, err := strconv.Atoi(os.Args[2])
	if err != nil {
		fmt.Printf("ERROR %s\n", err)
		return
	}
	maxRequestCount, err := strconv.Atoi(os.Args[3])
	sqsurl := os.Args[4]
	awsregion := os.Args[5]

	if err != nil {
		fmt.Printf("ERROR %s\n", err)
		return
	}
	client := &http.Client{}
	clientTimeout, _ := time.ParseDuration("1s")
	client.Timeout = clientTimeout
	fmt.Printf("Will spawn %d workers making %d requests to %s\n", concurrencycount, maxRequestCount, address)
	runLoadTest(client, sqsurl, address, maxRequestCount, concurrencycount, awsregion)
}

type Job struct{}

type RequestResult struct {
	Time             int64  `json:"time"`
	Host             string `json:"host"`
	Type             string `json:"type"`
	Status           int    `json:"status"`
	ElapsedFirstByte int64  `json:"elapsed-first-byte"`
	ElapsedLastByte  int64  `json:"elapsed-last-byte"`
	Elapsed          int64  `json:"elapsed"`
	Bytes            int    `json:"bytes"`
	State            string `json:"state"`
}

func runLoadTest(client *http.Client, sqsurl string, url string, totalRequests int, concurrencycount int, awsregion string) {
	sqsAdaptor := sqsadaptor.NewSQSAdaptor(sqsurl)
	jobs := make(chan Job, totalRequests)
	ch := make(chan RequestResult, totalRequests)
	var wg sync.WaitGroup
	var requestsSoFar int
	for i := 0; i < totalRequests; i++ {
		jobs <- Job{}
	}
	close(jobs)
	for i := 0; i < concurrencycount; i++ {
		wg.Add(1)
		go fetch(client, url, totalRequests, jobs, ch, &wg, awsregion)
	}
	fmt.Println("Waiting for results…")

	for requestsSoFar < totalRequests {
		var agg [100]RequestResult
		aggRequestCount := len(agg)
		i := 0

		var timeToFirstTotal int64
		var requestTimeTotal int64
		totBytesRead := 0
		statuses := make(map[string]int)
		var firstRequestTime int64
		var lastRequestTime int64
		var slowest int64
		var fastest int64
		for ; i < aggRequestCount && requestsSoFar < totalRequests; i++ {
			r := <-ch
			agg[i] = r
			requestsSoFar++
			if requestsSoFar%10 == 0 {
				fmt.Printf("\r%.2f%% done (%d requests out of %d)", (float64(requestsSoFar)/float64(totalRequests))*100.0, requestsSoFar, totalRequests)
			}
			if firstRequestTime == 0 {
				firstRequestTime = r.Time
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

			lastRequestTime = r.Time
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
		}
		durationSeconds := lastRequestTime - firstRequestTime
		if i == 0 {
			continue
		}
		if durationSeconds == 0 { // well…
			durationSeconds = 1
		}
		aggData := sqsadaptor.AggData{
			i,
			0, // totalTimedOut
			timeToFirstTotal / int64(i),
			totBytesRead,
			statuses,
			requestTimeTotal / int64(i),
			i / int(durationSeconds),
			slowest,
			fastest,
			awsregion,
		}
		sqsAdaptor.SendResult(aggData)
	}
	fmt.Printf("\nWaiting for workers…")
	wg.Wait()
	fmt.Printf("\nYay🎈  - %d requests completed\n", requestsSoFar)

}

func fetch(client *http.Client, address string, requestcount int, jobs <-chan Job, ch chan RequestResult, wg *sync.WaitGroup, awsregion string) {
	defer wg.Done()
	fmt.Printf("Fetching %s\n", address)
	for _ = range jobs {
		start := time.Now()
		req, err := http.NewRequest("GET", address, nil)
		req.Header.Add("User-Agent", "GOAD/0.1")
		response, err := client.Do(req)
		var status string
		var elapsedFirstByte time.Duration
		var elapsedLastByte time.Duration
		var elapsed time.Duration
		buf := []byte(" ")
		if err != nil {
			status = fmt.Sprintf("client.Do() failed: %s\n", err)
		} else {
			_, err = response.Body.Read(buf)
			if err != nil {
				status = fmt.Sprintf("reading first byte failed: %s\n", err)
			}
			elapsedFirstByte = time.Since(start)
			_, err = ioutil.ReadAll(response.Body)
			elapsedLastByte = time.Since(start)
			if err != nil {
				status = fmt.Sprintf("reading response body failed: %s\n", err)
			} else {
				status = "Success"
			}
			elapsed = time.Since(start)
		}
		result := RequestResult{
			start.Unix(),
			req.URL.Host,
			req.Method,
			response.StatusCode,
			elapsed.Nanoseconds(),
			elapsedFirstByte.Nanoseconds(),
			elapsedLastByte.Nanoseconds(),
			len(buf),
			status,
		}
		ch <- result
	}
}
