package main

import "github.com/gophergala2016/goad"

func main() {
	test := goad.NewTest(&goad.TestConfig{})
	test.Start()
}
