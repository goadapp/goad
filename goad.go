package goad

import (
	"errors"
	"fmt"
	"log"
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
	Region         string
}

const nano = 1000000000

func (c *TestConfig) cmd(sqsURL string) string {
	return fmt.Sprintf("./goad-lambda %s %d %d %s %s", c.URL, c.Concurrency, c.TotalRequests, sqsURL, c.Region)
}

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
	awsConfig := aws.NewConfig().WithRegion(t.config.Region)
	infra, err := infrastructure.New(awsConfig)
	if err != nil {
		log.Fatal(err)
	}

	t.invokeLambda(awsConfig, infra.QueueURL())

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

func (t *Test) invokeLambda(awsConfig *aws.Config, sqsURL string) {
	svc := lambda.New(session.New(), awsConfig)

	resp, err := svc.InvokeAsync(&lambda.InvokeAsyncInput{
		FunctionName: aws.String("goad"),
		InvokeArgs:   strings.NewReader(`{"cmd":"` + t.config.cmd(sqsURL) + `"}`),
	})
	fmt.Println(resp, err)
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
