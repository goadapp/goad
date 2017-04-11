package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"html"
	"math"
	"net"
	"net/http"
	"os"
	"strconv"
	"sync"
	"testing"
	"time"

	"github.com/aws/aws-sdk-go/service/lambda"
	"github.com/aws/aws-sdk-go/service/lambda/lambdaiface"
	"github.com/goadapp/goad/queue"
)

var port int
var urlStr string
var portStr string

func TestMain(m *testing.M) {
	port = 8080
	urlStr = fmt.Sprintf("http://localhost:%d/", port)
	portStr = fmt.Sprintf(":%d", port)
	code := m.Run()
	os.Exit(code)
}

func TestRequestMetric(t *testing.T) {
	metric := NewRequestMetric()
	agg := metric.aggregatedResults
	if agg.TotalReqs != 0 {
		t.Error("totalRequestsFinished should be initialized with 0")
	}
	if metric.requestCountSinceLastSend != 0 {
		t.Error("totalRequestsFinished should be initialized with 0")
	}
	if agg.Fastest != math.MaxInt64 {
		t.Error("Fastest should be initialized with a big value")
	}
	if agg.Slowest != 0 {
		t.Error("Slowest should be initialized with a big value")
	}
	if metric.firstRequestTime != 0 {
		t.Error("without requests this field should be 0")
	}
	if metric.lastRequestTime != 0 {
		t.Error("without requests this field should be 0")
	}
	if metric.timeToFirstTotal != 0 {
		t.Error("without requests this field should be 0")
	}
	if metric.requestTimeTotal != 0 {
		t.Error("without requests this field should be 0")
	}
	if agg.TotBytesRead != 0 {
		t.Error("without requests this field should be 0")
	}
}

func TestAddRequestStatus(t *testing.T) {
	success := 200
	successStr := strconv.Itoa(success)
	metric := NewRequestMetric()
	result := &requestResult{
		Status: success,
	}
	metric.addRequest(result)
	if metric.aggregatedResults.Statuses[successStr] != 1 {
		t.Error("metric should update cound of statuses map")
	}
	metric.addRequest(result)
	if metric.aggregatedResults.Statuses[successStr] != 2 {
		t.Error("metric should update cound of statuses map")
	}
}

func TestAddRequest(t *testing.T) {
	bytes := 1000
	elapsedFirst := int64(100)
	elapsedLast := int64(300)

	metric := NewRequestMetric()
	result := &requestResult{
		Time:             400,
		ElapsedFirstByte: elapsedFirst,
		ElapsedLastByte:  elapsedLast,
		Bytes:            bytes,
		Timeout:          false,
		ConnectionError:  false,
		Elapsed:          100,
	}
	metric.addRequest(result)
	if metric.lastRequestTime != result.Time+100 {
		t.Error("metrics should update lastRequestTime")
	}
	if metric.aggregatedResults.TotalReqs != 1 {
		t.Error("metrics should update totalRequestsFinished")
	}
	if metric.requestCountSinceLastSend != 1 {
		t.Error("metrics should upate totalRequestsFinished")
	}
	result.ElapsedLastByte = 400
	result.Time = 800
	metric.addRequest(result)
	agg := metric.aggregatedResults
	if agg.TotalReqs != 2 {
		t.Error("metrics should update totalRequestsFinished")
	}
	if metric.requestCountSinceLastSend != 2 {
		t.Error("metrics should upate totalRequestsFinished")
	}
	if agg.TotBytesRead != 2*bytes {
		t.Error("metrics should add successful requests Bytes to totalBytesRead")
	}
	if metric.requestTimeTotal != 700 {
		t.Error("metrics should add successful requests elapsedLast to requestTimeTotal")
	}
	if metric.timeToFirstTotal != 2*elapsedFirst {
		t.Error("metrics should add successful requests elapsedFirst to timeToFirstTotal")
	}
	if metric.firstRequestTime != 400 {
		t.Error("metrics should keep first requests time")
	}
	if metric.lastRequestTime != 800+100 {
		t.Error("metrics should update lastRequestsTime")
	}
	if agg.Fastest != 300 {
		t.Errorf("Expected fastes requests to have taken 300, was: %d", agg.Fastest)
	}
	if agg.Slowest != 400 {
		t.Errorf("Expected fastes requests to have taken 300, was: %d", agg.Fastest)
	}
	result.Timeout = true
	metric.addRequest(result)
	if agg.TotalReqs != 3 {
		t.Error("metrics should update totalRequestsFinished")
	}
	if metric.requestCountSinceLastSend != 3 {
		t.Error("metrics should upate totalRequestsFinished")
	}
	if agg.TotBytesRead != 2*bytes {
		t.Error("metrics should not add timedout requests Bytes to totalBytesRead")
	}
	if metric.requestTimeTotal != 700 {
		t.Error("metrics should not add timedout requests elapsedLast to requestTimeTotal")
	}
	if metric.timeToFirstTotal != 2*elapsedFirst {
		t.Error("metrics should not add timedout requests elapsedFirst to timeToFirstTotal")
	}
	if metric.firstRequestTime != 400 {
		t.Error("metrics should keep first requests time")
	}
	if metric.lastRequestTime != 800+100 {
		t.Error("metrics should update lastRequestsTime")
	}
	if agg.TotalTimedOut != 1 {
		t.Error("metrics should update TotalTimeOut")
	}
	result.ConnectionError = true
	result.Timeout = false
	metric.addRequest(result)
	if agg.TotalReqs != 4 {
		t.Error("metrics should update totalRequestsFinished")
	}
	if metric.requestCountSinceLastSend != 4 {
		t.Error("metrics should upate totalRequestsFinished")
	}
	if agg.TotBytesRead != 2*bytes {
		t.Error("metrics should not add timedout requests Bytes to totalBytesRead")
	}
	if metric.requestTimeTotal != 700 {
		t.Error("metrics should not add timedout requests elapsedLast to requestTimeTotal")
	}
	if metric.timeToFirstTotal != 2*elapsedFirst {
		t.Error("metrics should not add timedout requests elapsedFirst to timeToFirstTotal")
	}
	if metric.firstRequestTime != 400 {
		t.Error("metrics should keep first requests time")
	}
	if metric.lastRequestTime != 800+100 {
		t.Error("metrics should update lastRequestsTime")
	}
	if agg.TotalConnectionError != 1 {
		t.Error("metrics should update TotalConnectionError")
	}
}

func TestResetAndKeepTotalReqs(t *testing.T) {
	metric := NewRequestMetric()
	agg := metric.aggregatedResults
	agg.TotalReqs = 7
	metric.firstRequestTime = 123
	metric.lastRequestTime = 123
	metric.requestCountSinceLastSend = 123
	metric.requestTimeTotal = 123
	metric.timeToFirstTotal = 123
	agg.TotBytesRead = 123

	metric.resetAndKeepTotalReqs()
	agg = metric.aggregatedResults
	if agg.TotalReqs != 0 {
		t.Error("TotalReqs should be reset to 0")
	}
	if metric.requestCountSinceLastSend != 0 {
		t.Error("totalRequestsFinished should be reset to 0")
	}
	if agg.Fastest != math.MaxInt64 {
		t.Error("Fastest should be re-initialized with a big value")
	}
	if agg.Slowest != 0 {
		t.Error("Slowest should be re-initialized with a big value")
	}
	if metric.firstRequestTime != 0 {
		t.Error("firstRequestTime should be reset")
	}
	if metric.lastRequestTime != 0 {
		t.Error("lastRequestTime should be reset")
	}
	if metric.requestCountSinceLastSend != 0 {
		t.Error("requestCuntSinceLastSend should be reset")
	}
	if metric.requestTimeTotal != 0 {
		t.Error("requestTimeTotal should be reset")
	}
	if metric.timeToFirstTotal != 0 {
		t.Error("timeToFirstTotal should be reset")
	}
	if agg.TotBytesRead != 0 {
		t.Error("totalBytesRead should be reset")
	}
}

func TestMetricsAggregate(t *testing.T) {
	bytes := 1000
	elapsedFirst := int64(100)
	elapsedLast := int64(300)

	metric := NewRequestMetric()
	result := &requestResult{
		Time:             10000000,
		Elapsed:          10000000,
		ElapsedFirstByte: elapsedFirst,
		ElapsedLastByte:  elapsedLast,
		Bytes:            bytes,
		Timeout:          false,
		ConnectionError:  false,
	}
	metric.aggregate()
	agg := metric.aggregatedResults
	if agg.AveKBytesPerSec != 0 {
		t.Errorf("should result in 0, but was: %f", agg.AveKBytesPerSec)
	}
	if agg.AveReqPerSec != 0 {
		t.Errorf("should result in 0, but was: %f", agg.AveReqPerSec)
	}
	if agg.AveTimeToFirst != 0 {
		t.Errorf("should result in 0, but was: %d", agg.AveTimeToFirst)
	}
	if agg.AveTimeForReq != 0 {
		t.Errorf("should result in 0, but was: %d", agg.AveTimeForReq)
	}
	for i := 0; i < 10; i++ {
		result.Time += 10000000
		metric.addRequest(result)
	}
	metric.aggregate()
	if agg.AveKBytesPerSec != 100000.0 {
		t.Errorf("should result in average speed of 100000KB/s but was %f KB/s", agg.AveKBytesPerSec)
	}
	if agg.AveReqPerSec != 100 {
		t.Errorf("should result in average of 100 req/s but was %f req/s", agg.AveReqPerSec)
	}
	if agg.AveTimeToFirst != 100 {
		t.Errorf("should result in 100 but was %d", agg.AveTimeToFirst)
	}
	if agg.AveTimeForReq != 300 {
		t.Errorf("should result in 30 but was %d", agg.AveTimeForReq)
	}
	if agg.FatalError != "" {
		t.Errorf("there should be no fatal error but received: %s", agg.FatalError)
	}
	result.Timeout = true
	for i := 0; i < 10; i++ {
		result.Time += 10000000
		metric.addRequest(result)
	}
	if agg.FatalError != "" {
		t.Errorf("there should be no fatal error but received: %s", agg.FatalError)
	}
	metric.addRequest(result)
	if agg.FatalError != "Over 50% of requests failed, aborting" {
		t.Errorf("there should be a fatal error for failed requests but received: %s", agg.FatalError)
	}
}

type TestResultSender struct {
	sentResults []queue.AggData
}

func (s *TestResultSender) SendResult(data queue.AggData) {
	s.sentResults = append(s.sentResults, data)
}

type mockLambdaClient struct {
	lambdaiface.LambdaAPI
	input *invokeArgs
}

func (m *mockLambdaClient) InvokeAsync(in *lambda.InvokeAsyncInput) (*lambda.InvokeAsyncOutput, error) {
	buf := new(bytes.Buffer)
	buf.ReadFrom(in.InvokeArgs)
	args := &invokeArgs{}
	json.Unmarshal(buf.Bytes(), args)
	m.input = args
	return &lambda.InvokeAsyncOutput{}, nil
}

func TestQuitOnLambdaTimeout(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping test in short mode.")
	}
	handler := &delayRequstHandler{
		DelayMilliseconds: 400,
	}
	server := &testServer{
		Handler: handler,
	}
	server.Start()
	defer server.Stop()

	reportingFrequency := time.Duration(5) * time.Second
	settings := LambdaSettings{
		MaxRequestCount:          3,
		ConcurrencyCount:         1,
		ReportingFrequency:       reportingFrequency,
		StresstestTimeout:        10,
		LambdaExecTimeoutSeconds: 1,
		LambdaRegion:             "us-east-1",
	}
	settings.RequestParameters.URL = urlStr
	sender := &TestResultSender{}
	lambda := NewLambda(settings)
	lambda.resultSender = sender
	mockClient := &mockLambdaClient{}
	lambda.lambdaService = mockClient
	function := &lambdaTestFunction{
		lambda: lambda,
	}
	RunOrFailAfterTimout(t, function, 1400)
	resLength := len(sender.sentResults)
	timeoutRemaining := lambda.Settings.StresstestTimeout
	if timeoutRemaining != 9 {
		t.Errorf("we shoud have 9 seconds of stresstest left, actual: %d", timeoutRemaining)
	}
	requestCount := lambda.Settings.MaxRequestCount
	if requestCount != 3 {
		t.Errorf("the request count of 3 should not have changed, actual: %d", requestCount)
	}
	if resLength != 1 {
		t.Errorf("We should have received exactly 1 result but got %d instead.", resLength)
		t.FailNow()
	}
	if sender.sentResults[0].Finished == true {
		t.Error("lambda should not have finished all it's requests")
	}
	reqs := sender.sentResults[0].TotalReqs
	if reqs != 2 {
		t.Errorf("should have completed 2 requests yet but registered %d.", reqs)
	}
}

func TestMetricSendResults(t *testing.T) {
	bytes := 1024
	elapsedFirst := int64(100)
	elapsedLast := int64(300)
	result := &requestResult{
		Time:             400,
		ElapsedFirstByte: elapsedFirst,
		ElapsedLastByte:  elapsedLast,
		Bytes:            bytes,
		Timeout:          false,
		ConnectionError:  false,
	}

	metric := NewRequestMetric()
	sender := &TestResultSender{}

	metric.addRequest(result)
	metric.sendAggregatedResults(sender)
	if len(sender.sentResults) != 1 {
		t.Error("sender should have received one item")
		t.FailNow()
	}
}

func TestRunLoadTestWithHighConcurrency(t *testing.T) {
	server := createAndStartTestServer()
	defer server.Stop()

	runLoadTestWith(t, 500, 100, 500)
}

func TestRunLoadTestWithOneRequest(t *testing.T) {
	server := createAndStartTestServer()
	defer server.Stop()

	runLoadTestWith(t, 1, -1, 50)
}

func TestRunLoadTestWithZeroRequests(t *testing.T) {
	server := createAndStartTestServer()
	defer server.Stop()

	runLoadTestWith(t, 0, 1, 50)
}

func runLoadTestWith(t *testing.T, requestCount int, concurrency int, milliseconds int) {
	if testing.Short() {
		t.Skip("skipping test in short mode.")
	}
	reportingFrequency := time.Duration(5) * time.Second
	settings := LambdaSettings{
		MaxRequestCount:    requestCount,
		ConcurrencyCount:   concurrency,
		ReportingFrequency: reportingFrequency,
	}
	settings.RequestParameters.URL = urlStr
	sender := &TestResultSender{}
	lambda := NewLambda(settings)
	lambda.resultSender = sender
	function := &lambdaTestFunction{
		lambda: lambda,
	}
	RunOrFailAfterTimout(t, function, milliseconds)
	if len(sender.sentResults) != 1 {
		t.Error("sender should have received one item")
		t.FailNow()
	}
	results := sender.sentResults[0]
	if results.Finished != true {
		t.Error("the lambda should have finished it's results")
	}
	if results.TotalReqs != requestCount {
		t.Errorf("the lambda generated results for %d request, expected %d", results.TotalReqs, requestCount)
	}
}

func RunOrFailAfterTimout(t *testing.T, f TestFunction, milliseconds int) {
	timeout := time.Duration(milliseconds) * time.Millisecond
	select {
	case <-time.After(timeout):
		t.Error("Test is stuck")
		t.FailNow()
	case <-func() chan bool {
		quit := make(chan bool)
		go func() {
			f.Run()
			quit <- true
		}()
		return quit
	}():
	}
}

type lambdaTestFunction struct {
	lambda *goadLambda
}

type TestFunction interface {
	Run()
}

func (a *lambdaTestFunction) Run() {
	a.lambda.runLoadTest()
}

func TestFetchSuccess(t *testing.T) {
	handler := &requestCountHandler{}
	server := createAndStartTestServerWithHandler(handler)
	defer server.Stop()

	// setup for the fetch function
	expectedRequestCount := 1
	client := &http.Client{}
	r := requestParameters{
		URL: urlStr,
	}
	result := requestResult{}
	for result.State != "Success" {
		result = fetch(client, r, time.Now())
	}
	if handler.RequestCount != expectedRequestCount {
		t.Error("Did not receive exactly one request, received: ", handler.RequestCount)
	}
}

func createAndStartTestServer() *testServer {
	handler := &requestCountHandler{}
	server := createAndStartTestServerWithHandler(handler)
	return server
}

func createAndStartTestServerWithHandler(handler http.Handler) *testServer {
	server := &testServer{
		Handler: handler,
	}
	server.Start()
	return server
}

type testServer struct {
	Handler    http.Handler
	Listener   net.Listener
	HTTPServer http.Server
	wg         sync.WaitGroup
}

type requestCountHandler struct {
	RequestCount int
}

func (h *requestCountHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	h.RequestCount++
	fmt.Fprintf(w, "Hello, %q", html.EscapeString(r.URL.Path))
}

type delayRequstHandler struct {
	DelayMilliseconds int
}

func (h *delayRequstHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	time.Sleep(time.Duration(h.DelayMilliseconds) * time.Millisecond)
	fmt.Fprintf(w, "Hello, %q", html.EscapeString(r.URL.Path))
}

func (s *testServer) Start() {
	listener, err := net.Listen("tcp", portStr)
	if err != nil {
		panic(err)
	}
	s.Listener = listener
	s.HTTPServer = http.Server{
		Addr:    portStr,
		Handler: s.Handler,
	}
	s.wg.Add(1)
	go func() {
		s.HTTPServer.Serve(listener)
		s.wg.Done()
	}()
}

func (s *testServer) Stop() {
	s.Listener.Close()
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	s.HTTPServer.Shutdown(ctx)
	s.wg.Wait()
}

func _TestMultipleFetch(t *testing.T) {
	for i := 0; i < 1000; i++ {
		TestFetchSuccess(t)
	}
}
