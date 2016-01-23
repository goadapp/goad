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
	maxRequestCount, err := strconv.ParseInt(os.Args[3], 10, 64)
	sqsurl := os.Args[4]
	lambdainstance := os.Args[5]

	if err != nil {
		fmt.Printf("ERROR %s\n", err)
		return
	}
	client := &http.Client{}
	clientTimeout, _ := time.ParseDuration("1s")
	client.Timeout = clientTimeout
	fmt.Printf("Will spawn %d workers making %d requests to %s\n", concurrencycount, maxRequestCount, address)
	runLoadTest(client, sqsurl, address, maxRequestCount, concurrencycount, lambdainstance)
}

type Job struct{}

func runLoadTest(client *http.Client, sqsurl string, url string, totalRequests int64, concurrencycount int, lambdainstance string) {
	//sqsAdaptor := sqsadaptor.NewDummyAdaptor(sqsurl)
	jobs := make(chan Job, totalRequests)
	ch := make(chan sqsadaptor.Result, totalRequests)
	var wg sync.WaitGroup
	var requestsSoFar int64
	for i := int64(0); i < totalRequests; i++ {
		jobs <- Job{}
	}
	close(jobs)
	for i := 0; i < concurrencycount; i++ {
		wg.Add(1)
		go fetch(client, url, totalRequests, &requestsSoFar, jobs, ch, &wg, lambdainstance)
	}
	fmt.Println("Waiting for results…")

	for requestsSoFar < totalRequests {
		//r := <-ch
		_ = <-ch
		requestsSoFar++
		if requestsSoFar%10 == 0 {
			fmt.Printf("\r%.2f%% done (%d requests out of %d)", (float64(requestsSoFar)/float64(totalRequests))*100.0, requestsSoFar, totalRequests)
		}
		//sqsAdaptor.SendResult(r)
	}
	fmt.Printf("\nWaiting for workers…")
	wg.Wait()
	fmt.Printf("\nYay🎈  - %d requests completed\n", requestsSoFar)

}

func fetch(client *http.Client, address string, requestcount int64, requestsSoFar *int64, jobs <-chan Job, ch chan sqsadaptor.Result, wg *sync.WaitGroup, lambdainstance string) {
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
		result := sqsadaptor.Result{
			start.Format(time.RFC3339),
			req.URL.Host,
			req.Method,
			response.StatusCode,
			elapsed.Nanoseconds(),
			elapsedFirstByte.Nanoseconds(),
			elapsedLastByte.Nanoseconds(),
			len(buf),
			status,
			lambdainstance,
		}
		ch <- result
	}

}
