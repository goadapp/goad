package main

import (
	"fmt"
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
	if err != nil {
		fmt.Printf("ERROR %s\n", err)
		return
	}
	fmt.Printf("Will spawn %d workers each making %d requests to %s\n", concurrencycount, requestcount, address)
	runLoadTest(address, requestcount, concurrencycount)
}

func runLoadTest(url string, requestcount int, concurrencycount int) {
	totalRequests := requestcount * concurrencycount
	ch := make(chan string, totalRequests)
	var wg sync.WaitGroup
	for i := 0; i < concurrencycount; i++ {
		wg.Add(1)
		go fetch(url, requestcount, ch, &wg)
	}
	fmt.Println("Waiting for results…")

	completedRequests := 0
	for completedRequests < totalRequests {
		_ = <-ch
		completedRequests++
		fmt.Printf("\r%.2f%% done (%d requests out of %d)", (float64(completedRequests)/float64(totalRequests))*100.0, completedRequests, totalRequests)
	}
	wg.Wait()
	fmt.Printf("\nYay🎈\n")

}

func fetch(address string, requestcount int, ch chan string, wg *sync.WaitGroup) {
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
		//fmt.Printf("%s\n", string(contents))
		elapsed := time.Since(start)
		ch <- fmt.Sprintf("%d", elapsed.Nanoseconds())
	}

}
