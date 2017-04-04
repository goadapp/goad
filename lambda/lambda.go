package main

import (
	"bytes"
	"crypto/tls"
	"encoding/json"
	"flag"
	"fmt"
	"io/ioutil"
	"math"
	"net"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/lambda"
	"github.com/goadapp/goad/helpers"
	"github.com/goadapp/goad/queue"
	"github.com/goadapp/goad/version"
)

const AWS_MAX_TIMEOUT = 295

func main() {
	lambdaSettings := parseLambdaSettings()
	Lambda := NewLambda(lambdaSettings)
	Lambda.runLoadTest()
}

func parseLambdaSettings() LambdaSettings {
	var (
		address          string
		sqsurl           string
		concurrencycount int
		maxRequestCount  int
		execTimeout      int
		timeout          string
		frequency        string
		awsregion        string
		queueRegion      string
		requestMethod    string
		requestBody      string
		requestHeaders   helpers.StringsliceFlag
	)

	flag.StringVar(&address, "u", "", "URL to load test (required)")
	flag.StringVar(&requestMethod, "m", "GET", "HTTP method")
	flag.StringVar(&requestBody, "b", "", "HTTP request body")
	flag.StringVar(&awsregion, "r", "", "AWS region to run in")
	flag.StringVar(&queueRegion, "q", "", "Queue region")
	flag.StringVar(&sqsurl, "s", "", "sqsUrl")
	flag.StringVar(&timeout, "t", "15s", "request timeout in seconds")
	flag.StringVar(&frequency, "f", "15s", "Reporting frequency in seconds")

	flag.IntVar(&concurrencycount, "c", 10, "number of concurrent requests")
	flag.IntVar(&maxRequestCount, "n", 1000, "number of total requests to make")
	flag.IntVar(&execTimeout, "N", 0, "Maximum execution time in seconds")

	flag.Var(&requestHeaders, "H", "List of headers")
	flag.Parse()

	clientTimeout, _ := time.ParseDuration(timeout)
	fmt.Printf("Using a timeout of %s\n", clientTimeout)
	reportingFrequency, _ := time.ParseDuration(frequency)
	fmt.Printf("Using a reporting frequency of %s\n", reportingFrequency)

	fmt.Printf("Will spawn %d workers making %d requests to %s\n", concurrencycount, maxRequestCount, address)

	requestParameters := requestParameters{
		URL:            address,
		RequestHeaders: requestHeaders,
		RequestMethod:  requestMethod,
		RequestBody:    requestBody,
	}

	lambdaSettings := LambdaSettings{
		ClientTimeout:      clientTimeout,
		SqsURL:             sqsurl,
		AwsRegion:          awsregion,
		RequestCount:       maxRequestCount,
		ConcurrencyCount:   concurrencycount,
		QueueRegion:        queueRegion,
		ReportingFrequency: reportingFrequency,
		RequestParameters:  requestParameters,
		StresstestTimeout:  execTimeout,
	}
	return lambdaSettings
}

// LambdaSettings represent the Lambdas configuration
type LambdaSettings struct {
	LambdaExecTimeoutSeconds int
	SqsURL                   string
	RequestCount             int
	StresstestTimeout        int
	ConcurrencyCount         int
	QueueRegion              string
	ReportingFrequency       time.Duration
	ClientTimeout            time.Duration
	RequestParameters        requestParameters
	AwsRegion                string
}

// goadLambda holds the current state of the execution
type goadLambda struct {
	Settings     LambdaSettings
	HTTPClient   *http.Client
	Metrics      *requestMetric
	AwsConfig    *aws.Config
	resultSender resultSender
	results      chan requestResult
	jobs         chan struct{}
	StartTime    time.Time
	wg           sync.WaitGroup
}

type requestParameters struct {
	URL            string
	Requestcount   int
	RequestMethod  string
	RequestBody    string
	RequestHeaders []string
}

type requestResult struct {
	Time             int64  `json:"time"`
	Host             string `json:"host"`
	Type             string `json:"type"`
	Status           int    `json:"status"`
	ElapsedFirstByte int64  `json:"elapsed-first-byte"`
	ElapsedLastByte  int64  `json:"elapsed-last-byte"`
	Elapsed          int64  `json:"elapsed"`
	Bytes            int    `json:"bytes"`
	Timeout          bool   `json:"timeout"`
	ConnectionError  bool   `json:"connection-error"`
	State            string `json:"state"`
}

func (l *goadLambda) runLoadTest() {
	l.StartTime = time.Now()

	l.spawnConcurrentWorkers()

	ticker := time.NewTicker(l.Settings.ReportingFrequency)
	quit := time.NewTimer(time.Duration(l.Settings.LambdaExecTimeoutSeconds) * time.Second)
	timedOut := false

	for (l.Metrics.aggregatedResults.TotalReqs < l.Settings.RequestCount) && !timedOut {
		select {
		case r := <-l.results:
			l.Metrics.addRequest(&r)
			if l.Metrics.aggregatedResults.TotalReqs%1000 == 0 || l.Metrics.aggregatedResults.TotalReqs == l.Settings.RequestCount {
				fmt.Printf("\r%.2f%% done (%d requests out of %d)", (float64(l.Metrics.aggregatedResults.TotalReqs)/float64(l.Settings.RequestCount))*100.0, l.Metrics.aggregatedResults.TotalReqs, l.Settings.RequestCount)
			}
			continue

		case <-ticker.C:
			if l.Metrics.requestCountSinceLastSend > 0 {
				l.Metrics.sendAggregatedResults(l.resultSender)
				fmt.Printf("\nYay🎈  - %d requests completed\n", l.Metrics.aggregatedResults.TotalReqs)
			}
			continue

		case <-quit.C:
			ticker.Stop()
			timedOut = true
		}
	}
	if timedOut {
		fmt.Printf("-----------------timeout---------------------\n")
		l.forkNewLambda()
	} else {
		l.Metrics.aggregatedResults.Finished = true
	}
	l.Metrics.sendAggregatedResults(l.resultSender)
	fmt.Printf("\nYay🎈  - %d requests completed\n", l.Metrics.aggregatedResults.TotalReqs)
}

// NewLambda creates a new Lambda to execute a load test from a given
// LambdaSettings
func NewLambda(s LambdaSettings) *goadLambda {
	setLambdaExecTimeout(&s)
	setDefaultConcurrencyCount(&s)

	l := &goadLambda{}
	l.Settings = s

	l.Metrics = NewRequestMetric()
	l.setupHTTPClientForSelfsignedTLS()
	l.AwsConfig = l.setupAwsConfig()
	l.setupAwsSqsAdapter(l.AwsConfig)
	l.setupJobQueue()
	l.results = make(chan requestResult, l.Settings.RequestCount)
	return l
}

func setDefaultConcurrencyCount(s *LambdaSettings) {
	if s.ConcurrencyCount < 1 {
		s.ConcurrencyCount = 1
	}
}

func setLambdaExecTimeout(s *LambdaSettings) {
	if s.LambdaExecTimeoutSeconds <= 0 || s.LambdaExecTimeoutSeconds > AWS_MAX_TIMEOUT {
		s.LambdaExecTimeoutSeconds = AWS_MAX_TIMEOUT
	}
}

func (l *goadLambda) setupHTTPClientForSelfsignedTLS() {
	tr := &http.Transport{
		TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
	}
	l.HTTPClient = &http.Client{Transport: tr}
	l.HTTPClient.Timeout = l.Settings.ClientTimeout
}

func (l *goadLambda) setupAwsConfig() *aws.Config {
	return aws.NewConfig().WithRegion(l.Settings.QueueRegion)
}

func (l *goadLambda) setupAwsSqsAdapter(config *aws.Config) {
	l.resultSender = queue.NewSQSAdaptor(config, l.Settings.SqsURL)
}

func (l *goadLambda) setupJobQueue() {
	l.jobs = make(chan struct{}, l.Settings.RequestCount)
	for i := 0; i < l.Settings.RequestCount; i++ {
		l.jobs <- struct{}{}
	}
	close(l.jobs)
}

func (l *goadLambda) updateStresstestTimeout() {
	l.Settings.StresstestTimeout -= l.Settings.LambdaExecTimeoutSeconds
}

func (l *goadLambda) updateRemainingRequests() {
	l.Settings.RequestCount -= l.Metrics.aggregatedResults.TotalReqs
}

func (l *goadLambda) spawnConcurrentWorkers() {
	fmt.Print("Spawning workers…")
	for i := 0; i < l.Settings.ConcurrencyCount; i++ {
		l.spawnWorker()
		fmt.Print(".")
	}
	fmt.Println(" done.\nWaiting for results…")
}

func (l *goadLambda) spawnWorker() {
	l.wg.Add(1)
	go func() {
		defer l.wg.Done()
		work(l)
	}()
}

func work(l *goadLambda) {
	for {
		_, ok := <-l.jobs
		if !ok {
			break
		}
		l.results <- fetch(l.HTTPClient, l.Settings.RequestParameters, l.StartTime)
	}
}

func fetch(client *http.Client, p requestParameters, loadTestStartTime time.Time) requestResult {
	start := time.Now()
	req := prepareHttpRequest(p)
	response, err := client.Do(req)

	var status string
	var elapsedFirstByte time.Duration
	var elapsedLastByte time.Duration
	var elapsed time.Duration
	var statusCode int
	var bytesRead int
	buf := []byte(" ")
	timedOut := false
	connectionError := false
	isRedirect := err != nil && strings.Contains(err.Error(), "redirect")
	if err != nil && !isRedirect {
		status = fmt.Sprintf("ERROR: %s\n", err)
		switch err := err.(type) {
		case *url.Error:
			if err, ok := err.Err.(net.Error); ok && err.Timeout() {
				timedOut = true
			}
		case net.Error:
			if err.Timeout() {
				timedOut = true
			}
		}

		if !timedOut {
			connectionError = true
		}
	} else {
		statusCode = response.StatusCode
		elapsedFirstByte = time.Since(start)
		if !isRedirect {
			_, err = response.Body.Read(buf)
			firstByteRead := true
			if err != nil {
				status = fmt.Sprintf("reading first byte failed: %s\n", err)
				firstByteRead = false
			}
			body, err := ioutil.ReadAll(response.Body)
			if firstByteRead {
				bytesRead = len(body) + 1
			}
			elapsedLastByte = time.Since(start)
			if err != nil {
				// todo: detect timeout here as well
				status = fmt.Sprintf("reading response body failed: %s\n", err)
				connectionError = true
			} else {
				status = "Success"
			}
		} else {
			status = "Redirect"
		}
		response.Body.Close()

		elapsed = time.Since(start)
	}

	result := requestResult{
		Time:             start.Sub(loadTestStartTime).Nanoseconds(),
		Host:             req.URL.Host,
		Type:             req.Method,
		Status:           statusCode,
		ElapsedFirstByte: elapsedFirstByte.Nanoseconds(),
		ElapsedLastByte:  elapsedLastByte.Nanoseconds(),
		Elapsed:          elapsed.Nanoseconds(),
		Bytes:            bytesRead,
		Timeout:          timedOut,
		ConnectionError:  connectionError,
		State:            status,
	}
	return result
}

func prepareHttpRequest(params requestParameters) *http.Request {
	req, err := http.NewRequest(params.RequestMethod, params.URL, bytes.NewBufferString(params.RequestBody))
	if err != nil {
		fmt.Println("Error creating the HTTP request:", err)
		panic("")
	}
	req.Header.Add("Accept-Encoding", "gzip")
	for _, v := range params.RequestHeaders {
		header := strings.Split(v, ":")
		if strings.ToLower(strings.Trim(header[0], " ")) == "host" {
			req.Host = strings.Trim(header[1], " ")
		} else {
			req.Header.Add(strings.Trim(header[0], " "), strings.Trim(header[1], " "))
		}
	}

	if req.Header.Get("User-Agent") == "" {
		req.Header.Add("User-Agent", "Mozilla/5.0 (compatible; Goad/1.0; +https://goad.io)")
	}
	return req
}

type requestMetric struct {
	aggregatedResults         queue.AggData
	firstRequestTime          int64
	lastRequestTime           int64
	timeToFirstTotal          int64
	requestTimeTotal          int64
	totalBytesRead            int64
	requestCountSinceLastSend int64
}

type resultSender interface {
	SendResult(queue.AggData)
}

func NewRequestMetric() *requestMetric {
	metric := &requestMetric{}
	metric.resetAndKeepTotalReqs()
	return metric
}

func (m *requestMetric) addRequest(r *requestResult) {
	m.aggregatedResults.TotalReqs++
	m.requestCountSinceLastSend++
	if m.firstRequestTime == 0 {
		m.firstRequestTime = r.Time
	}
	m.lastRequestTime = r.Time + r.Elapsed

	if r.Timeout {
		m.aggregatedResults.TotalTimedOut++
	} else if r.ConnectionError {
		m.aggregatedResults.TotalConnectionError++
	} else {
		m.totalBytesRead += int64(r.Bytes)
		m.requestTimeTotal += r.ElapsedLastByte
		m.timeToFirstTotal += r.ElapsedFirstByte
		statusStr := strconv.Itoa(r.Status)
		_, ok := m.aggregatedResults.Statuses[statusStr]
		if !ok {
			m.aggregatedResults.Statuses[statusStr] = 1
		} else {
			m.aggregatedResults.Statuses[statusStr]++
		}
	}
	m.aggregate()
}

func (m *requestMetric) aggregate() {
	countOk := int(m.requestCountSinceLastSend) - (m.aggregatedResults.TotalTimedOut + m.aggregatedResults.TotalConnectionError)
	timeDelta := time.Duration(m.lastRequestTime-m.firstRequestTime) * time.Nanosecond
	timeDeltaInSeconds := float32(timeDelta.Seconds())
	if timeDeltaInSeconds > 0 {
		m.aggregatedResults.AveKBytesPerSec = float32(m.totalBytesRead) / timeDeltaInSeconds
		m.aggregatedResults.AveReqPerSec = float32(countOk) / timeDeltaInSeconds
	}
	if countOk > 0 {
		m.aggregatedResults.AveTimeToFirst = m.timeToFirstTotal / int64(countOk)
		m.aggregatedResults.AveTimeForReq = m.requestTimeTotal / int64(countOk)
	}
	m.aggregatedResults.FatalError = ""
	if (m.aggregatedResults.TotalTimedOut + m.aggregatedResults.TotalConnectionError) > int(m.requestCountSinceLastSend)/2 {
		m.aggregatedResults.FatalError = "Over 50% of requests failed, aborting"
	}
}

func (m *requestMetric) sendAggregatedResults(sender resultSender) {
	sender.SendResult(m.aggregatedResults)
	m.resetAndKeepTotalReqs()
}

func (m *requestMetric) resetAndKeepTotalReqs() {
	m.requestCountSinceLastSend = 0
	m.firstRequestTime = 0
	m.lastRequestTime = 0
	m.requestTimeTotal = 0
	m.timeToFirstTotal = 0
	m.totalBytesRead = 0
	saveTotalReqs := m.aggregatedResults.TotalReqs
	m.aggregatedResults = queue.AggData{
		Statuses:  make(map[string]int),
		Fastest:   math.MaxInt64,
		TotalReqs: saveTotalReqs,
		Finished:  false,
	}
}

func (l *goadLambda) forkNewLambda() {
	l.updateStresstestTimeout()
	l.updateRemainingRequests()
	svc := lambda.New(session.New(), l.AwsConfig)
	args := l.getInvokeArgsForFork()

	j, _ := json.Marshal(args)

	svc.InvokeAsync(&lambda.InvokeAsyncInput{
		FunctionName: aws.String("goad:" + version.LambdaVersion()),
		InvokeArgs:   bytes.NewReader(j),
	})
}

func (l *goadLambda) getInvokeArgsForFork() invokeArgs {
	args := newLambdaInvokeArgs()
	settings := l.Settings
	params := settings.RequestParameters
	args.Flags = []string{
		"-u",
		fmt.Sprintf("%s", params.URL),
		"-c",
		fmt.Sprintf("%s", strconv.Itoa(settings.ConcurrencyCount)),
		"-n",
		fmt.Sprintf("%s", strconv.Itoa(settings.RequestCount)),
		"-N",
		fmt.Sprintf("%s", strconv.Itoa(settings.StresstestTimeout)),
		"-s",
		fmt.Sprintf("%s", settings.SqsURL),
		"-q",
		fmt.Sprintf("%s", settings.AwsRegion),
		"-t",
		fmt.Sprintf("%s", settings.ClientTimeout.String()),
		"-f",
		fmt.Sprintf("%s", settings.ReportingFrequency.String()),
		"-r",
		fmt.Sprintf("%s", settings.AwsRegion),
		"-m",
		fmt.Sprintf("%s", params.RequestMethod),
		"-b",
		fmt.Sprintf("%s", params.RequestBody),
	}
	return args
}

type invokeArgs struct {
	File  string   `json:"file"`
	Flags []string `json:"args"`
}

func newLambdaInvokeArgs() invokeArgs {
	return invokeArgs{
		File: "./goad-lambda",
	}
}

// Min calculates minimum of two int64
func Min(x, y int64) int64 {
	if x < y {
		return x
	}
	return y
}

// Max calculates maximum of two int64
func Max(x, y int64) int64 {
	if x > y {
		return x
	}
	return y
}
