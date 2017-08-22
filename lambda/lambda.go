package main

import (
	"bytes"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
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
	"github.com/goadapp/goad/api"
	"github.com/goadapp/goad/infrastructure/aws/sqsadapter"
	"github.com/goadapp/goad/version"
	"github.com/streadway/amqp"
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
	runnerID                      = app.Flag("runner-id", "A id to identifiy this lambda function").Required().Int()
)

const AWS_MAX_TIMEOUT = 295

func main() {
	lambdaSettings := parseLambdaSettings()
	Lambda := newLambda(lambdaSettings)
	Lambda.runLoadTest()
}

func parseLambdaSettings() LambdaSettings {
	app.HelpFlag.Short('h')
	app.Version(version.String())
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
		RunnerID:              *runnerID,
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
	RunnerID                 int
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

// newLambda creates a new Lambda to execute a load test from a given
// LambdaSettings
func newLambda(s LambdaSettings) *goadLambda {
	setLambdaExecTimeout(&s)
	setDefaultConcurrencyCount(&s)

	l := &goadLambda{}
	l.Settings = s

	l.Metrics = NewRequestMetric(s.LambdaRegion, s.RunnerID)
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

func failOnError(err error, msg string) {
	if err != nil {
		log.Fatalf("%s: %s", msg, err)
		panic(fmt.Sprintf("%s: %s", msg, err))
	}
}

func (l *goadLambda) setupAwsSqsAdapter(config *aws.Config) {
	rabbit := os.ExpandEnv("$RABBITMQ")
	if rabbit != "" {
		l.resultSender = newRabbitMQAdapter(rabbit)
	} else {
		l.resultSender = sqsadapter.New(config, l.Settings.SqsURL)
	}
}

// RabbitMQAdapter to connect to RabbitMQ on docker daemon
type rabbitMQAdapter struct {
	resultSender
	ch   *amqp.Channel
	q    amqp.Queue
	conn *amqp.Connection
}

func (r *rabbitMQAdapter) SendResult(data api.RunnerResult) error {
	body, err := json.Marshal(data)
	if err != nil {
		panic(err)
	}
	err = r.ch.Publish(
		"",       // exchange
		r.q.Name, // routing key
		false,    // mandatory
		false,    // immediate
		amqp.Publishing{
			ContentType: "application/json",
			Body:        []byte(body),
		})
	log.Printf(" [x] Sent %s", body)
	return err
}

func newRabbitMQAdapter(queueURL string) resultSender {
	conn, err := amqp.Dial(queueURL)
	failOnError(err, "Failed to connect to RabbitMQ")
	ch, err := conn.Channel()
	failOnError(err, "Failed to open a channel")
	q, err := ch.QueueDeclare(
		"goad", // name
		false,  // durable
		false,  // delete when unused
		false,  // exclusive
		false,  // no-wait
		nil,    // arguments
	)
	failOnError(err, "Failed to declare a queue")
	return &rabbitMQAdapter{
		ch:   ch,
		q:    q,
		conn: conn,
	}
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
	aggregatedResults         *api.RunnerResult
	firstRequestTime          int64
	lastRequestTime           int64
	timeToFirstTotal          int64
	requestTimeTotal          int64
	requestCountSinceLastSend int64
}

type resultSender interface {
	SendResult(api.RunnerResult) error
}

func NewRequestMetric(region string, runnerID int) *requestMetric {
	metric := &requestMetric{
		aggregatedResults: &api.RunnerResult{
			Region:   region,
			RunnerID: runnerID,
		},
	}
	metric.resetAndKeepTotalReqs()
	return metric
}

func (m *requestMetric) addRequest(r *requestResult) {
	agg := m.aggregatedResults
	agg.RequestCount++
	m.requestCountSinceLastSend++
	if m.firstRequestTime == 0 {
		m.firstRequestTime = r.Time
	}
	m.lastRequestTime = r.Time + r.Elapsed

	if r.Timeout {
		agg.TimedOut++
	} else if r.ConnectionError {
		agg.ConnectionErrors++
	} else {
		agg.BytesRead += r.Bytes
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
	countOk := int(m.requestCountSinceLastSend) - (agg.TimedOut + agg.ConnectionErrors)
	agg.TimeDelta = time.Duration(m.lastRequestTime-m.firstRequestTime) * time.Nanosecond
	if countOk > 0 {
		agg.AveTimeToFirst = m.timeToFirstTotal / int64(countOk)
		agg.AveTimeForReq = m.requestTimeTotal / int64(countOk)
	}
	agg.FatalError = ""
	if (agg.TimedOut + agg.ConnectionErrors) > int(m.requestCountSinceLastSend)/2 {
		agg.FatalError = "Over 50% of requests failed, aborting"
	}
}

func (m *requestMetric) sendAggregatedResults(sender resultSender) {
	err := sender.SendResult(*m.aggregatedResults)
	failOnError(err, "Failed to send data to cli")
	m.resetAndKeepTotalReqs()
}

func (m *requestMetric) resetAndKeepTotalReqs() {
	m.requestCountSinceLastSend = 0
	m.firstRequestTime = 0
	m.lastRequestTime = 0
	m.requestTimeTotal = 0
	m.timeToFirstTotal = 0
	m.aggregatedResults = &api.RunnerResult{
		Region:   m.aggregatedResults.Region,
		RunnerID: m.aggregatedResults.RunnerID,
		Statuses: make(map[string]int),
		Fastest:  math.MaxInt64,
		Finished: false,
	}
}

func (l *goadLambda) forkNewLambda() {
	svc := l.provideLambdaService()
	args := l.getInvokeArgsForFork()

	j, _ := json.Marshal(args)

	output, err := svc.Invoke(&lambda.InvokeInput{
		FunctionName: aws.String("goad"),
		Payload:      j,
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
		fmt.Sprintf("--concurrency=%s", strconv.Itoa(settings.ConcurrencyCount)),
		fmt.Sprintf("--requests=%s", strconv.Itoa(settings.MaxRequestCount)),
		fmt.Sprintf("--completed-count=%s", strconv.Itoa(l.Settings.CompletedRequestCount)),
		fmt.Sprintf("--execution-time=%s", strconv.Itoa(settings.StresstestTimeout)),
		fmt.Sprintf("--sqsurl=%s", settings.SqsURL),
		fmt.Sprintf("--queue-region=%s", settings.QueueRegion),
		fmt.Sprintf("--client-timeout=%s", settings.ClientTimeout),
		fmt.Sprintf("--runner-id=%d", settings.RunnerID),
		fmt.Sprintf("--frequency=%s", settings.ReportingFrequency),
		fmt.Sprintf("--aws-region=%s", settings.LambdaRegion),
		fmt.Sprintf("--method=%s", settings.RequestParameters.RequestMethod),
		fmt.Sprintf("--body=%s", settings.RequestParameters.RequestBody),
	}
	args.Flags = append(args.Flags, fmt.Sprintf("%s", params.URL))
	fmt.Println(args.Flags)
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
