package main

import (
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"time"
)

func main() {
	for _, arg := range os.Args[1:] {
		fmt.Println("address was " + arg)
		fetch(arg)
	}
}

func fetch(arg string) {
	start := time.Now()
	response, err := http.Get(arg)
	contents, err := ioutil.ReadAll(response.Body)
	if err != nil {
		fmt.Printf("ERROR %s\n", err)
	} else {
		fmt.Printf("%s\n", string(contents))
	}
	elapsed := time.Since(start)
	fmt.Printf("%dnS\n", elapsed.Nanoseconds())
}
