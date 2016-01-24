package main

import (
	"flag"
	"fmt"
	"os"
	"time"

	"github.com/gophergala2016/goad"
)

var (
	url         string
	concurrency uint
	requests    uint
	timeout     uint
	region      string
)

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
	test.Start()
}
