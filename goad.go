package goad

import (
	"errors"
	"fmt"
	"log"
	"math"
	"strings"
	"time"

	"github.com/gophergala2016/goad/Godeps/_workspace/src/github.com/aws/aws-sdk-go/aws"
	"github.com/gophergala2016/goad/Godeps/_workspace/src/github.com/aws/aws-sdk-go/aws/session"
	"github.com/gophergala2016/goad/Godeps/_workspace/src/github.com/aws/aws-sdk-go/service/lambda"
	"github.com/gophergala2016/goad/infrastructure"
	"github.com/gophergala2016/goad/queue"
)

// TestConfig type
type TestConfig struct {
	URL            string
	Concurrency    uint
	TotalRequests  uint
	RequestTimeout time.Duration
	Regions        []string
}

const nano = 1000000000

// Test type
type Test struct {
	config *TestConfig
}

// NewTest returns a configured Test
func NewTest(config *TestConfig) (*Test, error) {
	err := config.check()
	if err != nil {
		return nil, err
	}
	return &Test{config}, nil
}

// Start a test
func (t *Test) Start() <-chan queue.RegionsAggData {
	awsConfig := aws.NewConfig().WithRegion(t.config.Regions[0])
	infra, err := infrastructure.New(t.config.Regions, awsConfig)
	if err != nil {
		log.Fatal(err)
	}

	t.invokeLambdas(awsConfig, infra.QueueURL())

	results := make(chan queue.RegionsAggData)

	go func() {
		for result := range queue.Aggregate(awsConfig, infra.QueueURL(), t.config.TotalRequests) {
			results <- result
		}
		infra.Clean()
		close(results)
	}()

	return results
}

func (t *Test) invokeLambdas(awsConfig *aws.Config, sqsURL string) {
	lambdas := numberOfLambdas(t.config.Concurrency)

	for i := 0; i < lambdas; i++ {
		region := t.config.Regions[i%len(t.config.Regions)]
		requests, requestsRemainder := divide(t.config.TotalRequests, lambdas)
		concurrency, _ := divide(t.config.Concurrency, lambdas)

		if requestsRemainder > 0 && i == lambdas-1 {
			requests += requestsRemainder
		}

		c := t.config
		cmd := fmt.Sprintf("./goad-lambda %s %d %d %s %s %s %s %s",
			c.URL, concurrency, requests, sqsURL, region, c.RequestTimeout,
			reportingFrequency(lambdas), c.Regions[0])

		config := aws.NewConfig().WithRegion(region)
		go t.invokeLambda(config, cmd)
	}
}

func (t *Test) invokeLambda(awsConfig *aws.Config, cmd string) {
	svc := lambda.New(session.New(), awsConfig)

	_, err := svc.InvokeAsync(&lambda.InvokeAsyncInput{
		FunctionName: aws.String("goad"),
		InvokeArgs:   strings.NewReader(`{"cmd":"` + cmd + `"}`),
	})

	if err != nil {
		log.Fatal(err)
	}
}

func numberOfLambdas(concurrency uint) int {
	if concurrency/10 > 100 {
		return 100
	}
	return int(concurrency-1)/10 + 1
}

func divide(dividend uint, divisor int) (quotient, remainder uint) {
	return dividend / uint(divisor), dividend % uint(divisor)
}

func reportingFrequency(numberOfLambdas int) time.Duration {
	return time.Duration((math.Log2(float64(numberOfLambdas)) + 1)) * time.Second
}

func (c TestConfig) check() error {
	if c.Concurrency < 1 || c.Concurrency > 100000 {
		return errors.New("Invalid concurrency (use 1 - 100000)")
	}
	if c.TotalRequests < 1 || c.TotalRequests > 1000000 {
		return errors.New("Invalid total requests (use 1 - 1000000)")
	}
	if c.RequestTimeout.Nanoseconds() < nano || c.RequestTimeout.Nanoseconds() > nano*100 {
		return errors.New("Invalid timeout (1s - 100s)")
	}
	return nil
}
