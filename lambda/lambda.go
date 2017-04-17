package main

import (
	"bytes"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"math"
	"net"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"

	kingpin "gopkg.in/alecthomas/kingpin.v2"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/lambda"
	"github.com/aws/aws-sdk-go/service/lambda/lambdaiface"
	"github.com/goadapp/goad/queue"
	"github.com/goadapp/goad/version"
)

var (
	app = kingpin.New("goad-lambda", "Utility deployed into aws lambda by goad")

	address        = app.Arg("url", "URL to load test").Required().String()
	requestMethod  = app.Flag("method", "HTTP method").Short('m').Default("GET").String()
	requestBody    = app.Flag("body", "HTTP request body").Short('b').String()
	requestHeaders = app.Flag("header", "List of headers").Short('H').Strings()

	awsRegion   = app.Flag("aws-region", "AWS region to run in").Short('r').String()
	queueRegion = app.Flag("queue-region", "SQS queue region").Short('q').String()
	sqsURL      = app.Flag("sqsurl", "SQS URL").String()

	clientTimeout      = app.Flag("client-timeout", "Request timeout duration").Short('s').Default("15s").Duration()
	reportingFrequency = app.Flag("frequency", "Reporting frequency in seconds").Short('f').Default("15s").Duration()

	concurrencyCount              = app.Flag("concurrency", "Number of concurrent requests").Short('c').Default("10").Int()
	maxRequestCount               = app.Flag("requests", "Total number of requests to make").Short('n').Default("10").Int()
	previousCompletedRequestCount = app.Flag("completed-count", "Number of requests already completed in case of lambda timeout").Short('p').Default("0").Int()
	execTimeout                   = app.Flag("execution-time", "Maximum execution time in seconds").Short('t').Default("0").Int()
)

const AWS_MAX_TIMEOUT = 295

func main() {
	lambdaSettings := parseLambdaSettings()
	Lambda := NewLambda(lambdaSettings)
	Lambda.runLoadTest()
}

func parseLambdaSettings() LambdaSettings {
	app.HelpFlag.Short('h')
	app.Version(version.Version)
	kingpin.MustParse(app.Parse(os.Args[1:]))

	requestParameters := requestParameters{
		URL:            *address,
		RequestHeaders: *requestHeaders,
		RequestMethod:  *requestMethod,
		RequestBody:    *requestBody,
	}

	lambdaSettings := LambdaSettings{
		ClientTimeout:         *clientTimeout,
		SqsURL:                *sqsURL,
		MaxRequestCount:       *maxRequestCount,
		CompletedRequestCount: *previousCompletedRequestCount,
		ConcurrencyCount:      *concurrencyCount,
		QueueRegion:           *queueRegion,
		LambdaRegion:          *awsRegion,
		ReportingFrequency:    *reportingFrequency,
		RequestParameters:     requestParameters,
		StresstestTimeout:     *execTimeout,
	}
	return lambdaSettings
}

// LambdaSettings represent the Lambdas configuration
type LambdaSettings struct {
	LambdaExecTimeoutSeconds int
	SqsURL                   string
	MaxRequestCount          int
	CompletedRequestCount    int
	StresstestTimeout        int
	ConcurrencyCount         int
	QueueRegion              string
	LambdaRegion             string
	ReportingFrequency       time.Duration
	ClientTimeout            time.Duration
	RequestParameters        requestParameters
}

// goadLambda holds the current state of the execution
type goadLambda struct {
	Settings      LambdaSettings
	HTTPClient    *http.Client
	Metrics       *requestMetric
	lambdaService lambdaiface.LambdaAPI
	resultSender  resultSender
	results       chan requestResult
	jobs          chan struct{}
	StartTime     time.Time
	wg            sync.WaitGroup
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
	fmt.Printf("Using a timeout of %s\n", l.Settings.ClientTimeout)
	fmt.Printf("Using a reporting frequency of %s\n", l.Settings.ReportingFrequency)
	fmt.Printf("Will spawn %d workers making %d requests to %s\n", l.Settings.ConcurrencyCount, l.Settings.MaxRequestCount, l.Settings.RequestParameters.URL)

	l.StartTime = time.Now()

	l.spawnConcurrentWorkers()

	ticker := time.NewTicker(l.Settings.ReportingFrequency)
	quit := time.NewTimer(time.Duration(l.Settings.LambdaExecTimeoutSeconds) * time.Second)
	timedOut := false
	finished := false

	for !timedOut && !finished {
		select {
		case r := <-l.results:
			l.Settings.CompletedRequestCount++

			l.Metrics.addRequest(&r)
			if l.Settings.CompletedRequestCount%1000 == 0 || l.Settings.CompletedRequestCount == l.Settings.MaxRequestCount {
				fmt.Printf("\r%.2f%% done (%d requests out of %d)", (float64(l.Settings.CompletedRequestCount)/float64(l.Settings.MaxRequestCount))*100.0, l.Settings.CompletedRequestCount, l.Settings.MaxRequestCount)
			}
			continue

		case <-ticker.C:
			if l.Metrics.requestCountSinceLastSend > 0 {
				l.Metrics.sendAggregatedResults(l.resultSender)
				fmt.Printf("\nYay🎈  - %d requests completed\n", l.Settings.CompletedRequestCount)
			}
			continue

		case <-func() chan bool {
			quit := make(chan bool)
			go func() {
				l.wg.Wait()
				quit <- true
			}()
			return quit
		}():
			finished = true
			continue

		case <-quit.C:
			ticker.Stop()
			timedOut = true
			finished = l.updateStresstestTimeout()
		}
	}
	if timedOut && !finished {
		l.forkNewLambda()
	}
	l.Metrics.aggregatedResults.Finished = finished
	l.Metrics.sendAggregatedResults(l.resultSender)
	fmt.Printf("\nYay🎈  - %d requests completed\n", l.Settings.CompletedRequestCount)
}

// NewLambda creates a new Lambda to execute a load test from a given
// LambdaSettings
func NewLambda(s LambdaSettings) *goadLambda {
	setLambdaExecTimeout(&s)
	setDefaultConcurrencyCount(&s)

	l := &goadLambda{}
	l.Settings = s

	l.Metrics = NewRequestMetric(s.LambdaRegion)
	remainingRequestCount := s.MaxRequestCount - s.CompletedRequestCount
	if remainingRequestCount < 0 {
		remainingRequestCount = 0
	}
	l.setupHTTPClientForSelfsignedTLS()
	awsSqsConfig := l.setupAwsConfig()
	l.setupAwsSqsAdapter(awsSqsConfig)
	l.setupJobQueue(remainingRequestCount)
	l.results = make(chan requestResult)
	return l
}

func setDefaultConcurrencyCount(s *LambdaSettings) {
	if s.ConcurrencyCount < 1 {
		s.ConcurrencyCount = 1
	}
}

func setLambdaExecTimeout(s *LambdaSettings) {
	if s.StresstestTimeout <= 0 || s.StresstestTimeout > AWS_MAX_TIMEOUT {
		s.LambdaExecTimeoutSeconds = AWS_MAX_TIMEOUT
	} else {
		s.LambdaExecTimeoutSeconds = s.StresstestTimeout
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

func (l *goadLambda) setupJobQueue(count int) {
	l.jobs = make(chan struct{}, count)
	for i := 0; i < count; i++ {
		l.jobs <- struct{}{}
	}
	close(l.jobs)
}

func (l *goadLambda) updateStresstestTimeout() bool {
	if l.Settings.StresstestTimeout != 0 {
		l.Settings.StresstestTimeout -= l.Settings.LambdaExecTimeoutSeconds
		return l.Settings.StresstestTimeout <= 0
	}
	return false
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
		if l.Settings.MaxRequestCount > 0 {
			_, ok := <-l.jobs
			if !ok {
				break
			}
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
	aggregatedResults         *queue.AggData
	firstRequestTime          int64
	lastRequestTime           int64
	timeToFirstTotal          int64
	requestTimeTotal          int64
	requestCountSinceLastSend int64
}

type resultSender interface {
	SendResult(queue.AggData)
}

func NewRequestMetric(region string) *requestMetric {
	metric := &requestMetric{
		aggregatedResults: &queue.AggData{Region: region},
	}
	metric.resetAndKeepTotalReqs()
	return metric
}

func (m *requestMetric) addRequest(r *requestResult) {
	agg := m.aggregatedResults
	agg.TotalReqs++
	m.requestCountSinceLastSend++
	if m.firstRequestTime == 0 {
		m.firstRequestTime = r.Time
	}
	m.lastRequestTime = r.Time + r.Elapsed

	if r.Timeout {
		agg.TotalTimedOut++
	} else if r.ConnectionError {
		agg.TotalConnectionError++
	} else {
		agg.TotBytesRead += r.Bytes
		m.requestTimeTotal += r.ElapsedLastByte
		m.timeToFirstTotal += r.ElapsedFirstByte

		agg.Fastest = Min(r.ElapsedLastByte, agg.Fastest)
		agg.Slowest = Max(r.ElapsedLastByte, agg.Slowest)

		statusStr := strconv.Itoa(r.Status)
		_, ok := agg.Statuses[statusStr]
		if !ok {
			agg.Statuses[statusStr] = 1
		} else {
			agg.Statuses[statusStr]++
		}
	}
	m.aggregate()
}

func (m *requestMetric) aggregate() {
	agg := m.aggregatedResults
	countOk := int(m.requestCountSinceLastSend) - (agg.TotalTimedOut + agg.TotalConnectionError)
	timeDelta := time.Duration(m.lastRequestTime-m.firstRequestTime) * time.Nanosecond
	timeDeltaInSeconds := float32(timeDelta.Seconds())
	if timeDeltaInSeconds > 0 {
		agg.AveKBytesPerSec = float32(agg.TotBytesRead) / timeDeltaInSeconds
		agg.AveReqPerSec = float32(countOk) / timeDeltaInSeconds
	}
	if countOk > 0 {
		agg.AveTimeToFirst = m.timeToFirstTotal / int64(countOk)
		agg.AveTimeForReq = m.requestTimeTotal / int64(countOk)
	}
	agg.FatalError = ""
	if (agg.TotalTimedOut + agg.TotalConnectionError) > int(m.requestCountSinceLastSend)/2 {
		agg.FatalError = "Over 50% of requests failed, aborting"
	}
}

func (m *requestMetric) sendAggregatedResults(sender resultSender) {
	sender.SendResult(*m.aggregatedResults)
	m.resetAndKeepTotalReqs()
}

func (m *requestMetric) resetAndKeepTotalReqs() {
	m.requestCountSinceLastSend = 0
	m.firstRequestTime = 0
	m.lastRequestTime = 0
	m.requestTimeTotal = 0
	m.timeToFirstTotal = 0
	m.aggregatedResults = &queue.AggData{
		Region:   m.aggregatedResults.Region,
		Statuses: make(map[string]int),
		Fastest:  math.MaxInt64,
		Finished: false,
	}
}

func (l *goadLambda) forkNewLambda() {
	svc := l.provideLambdaService()
	args := l.getInvokeArgsForFork()

	j, _ := json.Marshal(args)

	output, err := svc.InvokeAsync(&lambda.InvokeAsyncInput{
		FunctionName: aws.String("goad:" + version.LambdaVersion()),
		InvokeArgs:   bytes.NewReader(j),
	})
	fmt.Println(output)
	fmt.Println(err)
}

func (l *goadLambda) provideLambdaService() lambdaiface.LambdaAPI {
	if l.lambdaService == nil {
		l.lambdaService = lambda.New(session.New(), aws.NewConfig().WithRegion(l.Settings.LambdaRegion))
	}
	return l.lambdaService
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
		fmt.Sprintf("%s", strconv.Itoa(settings.MaxRequestCount)),
		"-p",
		fmt.Sprintf("%s", strconv.Itoa(l.Settings.CompletedRequestCount)),
		"-N",
		fmt.Sprintf("%s", strconv.Itoa(settings.StresstestTimeout)),
		"-s",
		fmt.Sprintf("%s", settings.SqsURL),
		"-q",
		fmt.Sprintf("%s", settings.QueueRegion),
		"-t",
		fmt.Sprintf("%s", settings.ClientTimeout.String()),
		"-f",
		fmt.Sprintf("%s", settings.ReportingFrequency.String()),
		"-r",
		fmt.Sprintf("%s", settings.LambdaRegion),
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
