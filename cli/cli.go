package main

import (
	"time"

	"github.com/gophergala2016/goad"
)

func main() {
	test := goad.NewTest(&goad.TestConfig{
		URL:            "http://dll.nu/",
		Concurrency:    50,
		TotalRequests:  1000,
		RequestTimeout: time.Second,
		Region:         "eu-west-1",
	})
	test.Start()
}
