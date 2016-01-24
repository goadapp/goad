package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/gophergala2016/goad"
	"github.com/gophergala2016/goad/Godeps/_workspace/src/github.com/gorilla/websocket"
	"github.com/gophergala2016/goad/queue"
)

var addr = flag.String("addr", ":8080", "http service address")
var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool {
		return true
	},
}

func main() {
	Serve()
}

func jsonFromRegionsAggData(result queue.RegionsAggData) (string, error) {
	data, jsonerr := json.Marshal(result)
	if jsonerr != nil {
		return "", jsonerr
	}
	return string(data), nil
}

func serveResults(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/goad" {
		http.Error(w, "Not found", 404)
		return
	}
	if r.Method != "GET" {
		http.Error(w, "Method not allowed", 405)
		return
	}
	w.Header().Set("Content-Type", "application/json; charset=utf-8")

	url := r.URL.Query().Get("url")
	if len(url) == 0 {
		http.Error(w, "Missing URL", 400)
		return
	}

	// Commented out param handling, we have hard coded limits for the website testing
	//
	// concurrencyStr := r.URL.Query().Get("c")
	// concurrency, cerr := strconv.Atoi(concurrencyStr)
	// if cerr != nil {
	// 	http.Error(w, "Invalid concurrency", 400)
	// 	return
	// }
	//
	// totStr := r.URL.Query().Get("tot")
	// tot, toterr := strconv.Atoi(totStr)
	// if toterr != nil {
	// 	http.Error(w, "Invalid total", 400)
	// 	return
	// }
	//
	// timeoutStr := r.URL.Query().Get("timeout")
	// timeout, timeouterr := strconv.Atoi(timeoutStr)
	// if timeouterr != nil {
	// 	http.Error(w, "Invalid timeout", 400)
	// 	return
	// }

	c, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Print("Websocket upgrade:", err)
		return
	}
	defer c.Close()

	config := goad.TestConfig{
		url,
		5,
		1000,
		time.Duration(7),
		"us-east-1",
	}

	test, testerr := goad.NewTest(&config)
	if testerr != nil {
		fmt.Println(testerr)
		return
	}
	resultChan := test.Start()

	for result := range resultChan {
		message, jsonerr := jsonFromRegionsAggData(result)
		if jsonerr != nil {
			log.Println(jsonerr)
			break
		}
		go readLoop(c)
		err = c.WriteMessage(websocket.TextMessage, []byte(message))
		if err != nil {
			log.Println("write:", err)
			break
		}
	}
}

func readLoop(c *websocket.Conn) {
	for {
		if _, _, err := c.NextReader(); err != nil {
			c.Close()
			break
		}
	}
}

func health(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
}

// Serve waits for connections and serves the results
func Serve() {
	http.HandleFunc("/goad", serveResults)
	http.HandleFunc("/_health", health)
	err := http.ListenAndServe(*addr, nil)
	if err != nil {
		log.Fatal("ListenAndServe: ", err)
	}
}
