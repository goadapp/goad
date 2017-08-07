package main

import (
	"encoding/json"
	"flag"
	"log"
	"net/http"
	"strconv"

	"github.com/goadapp/goad/goad"
	"github.com/goadapp/goad/result"
	"github.com/gorilla/websocket"
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

func jsonFromRegionsAggData(result *result.LambdaResults) (string, error) {
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

	concurrencyStr := r.URL.Query().Get("concurrency")
	concurrency, concurrencyerr := strconv.Atoi(concurrencyStr)
	if concurrencyerr != nil {
		http.Error(w, "Invalid concurrency", 400)
		return
	}

	requestsStr := r.URL.Query().Get("requests")
	requests, requestserr := strconv.Atoi(requestsStr)
	if requestserr != nil {
		http.Error(w, "Invalid number of requests", 400)
		return
	}

	timelimitStr := r.URL.Query().Get("timelimit")
	timelimit, timelimiterr := strconv.Atoi(timelimitStr)
	if timelimiterr != nil {
		http.Error(w, "Invalid timelimit", 400)
		return
	}

	timeoutStr := r.URL.Query().Get("timeout")
	timeout, timeouterr := strconv.Atoi(timeoutStr)
	if timeouterr != nil {
		http.Error(w, "Invalid timeout", 400)
		return
	}

	regions := r.URL.Query()["region[]"]
	if len(regions) == 0 {
		http.Error(w, "Missing region", 400)
		return
	}

	config := goad.TestConfig{
		URL:         url,
		Concurrency: concurrency,
		Requests:    requests,
		Timelimit:   timelimit,
		Timeout:     timeout,
		Regions:     regions,
		Method:      "GET",
	}

	test, testerr := goad.NewTest(&config)
	if testerr != nil {
		http.Error(w, testerr.Error(), 400)
		return
	}

	c, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Print("Websocket upgrade:", err)
		return
	}
	defer c.Close()

	resultChan, teardown := test.Start()
	defer teardown()

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
