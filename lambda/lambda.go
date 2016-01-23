package main

import (
	"fmt"
	"github.com/gophergala2016/goad/lambda/sqsadaptor"
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
	requestcount, err := strconv.Atoi(os.Args[3])
	sqsurl := os.Args[4]
	if err != nil {
		fmt.Printf("ERROR %s\n", err)
		return
	}
	fmt.Printf("Will spawn %d workers each making %d requests to %s\n", concurrencycount, requestcount, address)
	runLoadTest(sqsurl, address, requestcount, concurrencycount)
}

func runLoadTest(sqsurl string, url string, requestcount int, concurrencycount int) {
	sqsAdaptor := sqsadaptor.NewDummyAdaptor(sqsurl)

	totalRequests := requestcount * concurrencycount
	ch := make(chan sqsadaptor.Result, totalRequests)
	var wg sync.WaitGroup
	for i := 0; i < concurrencycount; i++ {
		wg.Add(1)
		go fetch(url, requestcount, ch, &wg)
	}
	fmt.Println("Waiting for results…")

	completedRequests := 0
	for completedRequests < totalRequests {
		r := <-ch
		completedRequests++
		if completedRequests%10 == 0 {
			fmt.Printf("\r%.2f%% done (%d requests out of %d)", (float64(completedRequests)/float64(totalRequests))*100.0, completedRequests, totalRequests)
		}
		sqsAdaptor.SendResult(r)
	}
	wg.Wait()
	fmt.Printf("\nYay🎈\n")

}

func fetch(address string, requestcount int, ch chan sqsadaptor.Result, wg *sync.WaitGroup) {
	defer wg.Done()
	fmt.Printf("Fetching %s %d times\n", address, requestcount)
	for i := 0; i < requestcount; i++ {
		start := time.Now()
		response, err := http.Get(address)
		if err != nil {
			fmt.Printf("ERROR %s\n", err)
			return
		}
		_, err = ioutil.ReadAll(response.Body)
		if err != nil {
			fmt.Printf("ERROR %s\n", err)
			return
		}
		elapsed := time.Since(start)
		result := sqsadaptor.Result{
			"2016-01-01 10:00:00",
			"example.com",
			"Fetch",
			"928429348",
			200,
			elapsed.Nanoseconds(),
			2398,
			"Finished",
		}
		ch <- result
	}

}
