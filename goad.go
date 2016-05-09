package goad

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"math"
	"strconv"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/lambda"
	"github.com/goadapp/goad/infrastructure"
	"github.com/goadapp/goad/queue"
	"github.com/goadapp/goad/version"
)

// TestConfig type
type TestConfig struct {
	URL            string
	Concurrency    uint
	TotalRequests  uint
	RequestTimeout time.Duration
	Regions        []string
	Method         string
	Body           string
	Headers        []string
	AwsProfile     string
}

type invokeArgs struct {
	File string   `json:"file"`
	Args []string `json:"args"`
}

const nano = 1000000000

var supportedRegions = []string{
	"us-east-1",
	"us-west-2",
	"eu-west-1",
	"ap-northeast-1",
	"eu-central-1",
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
	awsConfig := aws.NewConfig().WithRegion(t.config.Regions[0])

	if t.config.AwsProfile != "" {
		creds := credentials.NewSharedCredentials("", t.config.AwsProfile)
		if _, err := creds.Get(); err != nil {
			log.Fatal(err)
		}
		awsConfig.WithCredentials(creds)
	}

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
	lambdas := numberOfLambdas(t.config.Concurrency, len(t.config.Regions))

	for i := 0; i < lambdas; i++ {
		region := t.config.Regions[i%len(t.config.Regions)]
		requests, requestsRemainder := divide(t.config.TotalRequests, lambdas)
		concurrency, _ := divide(t.config.Concurrency, lambdas)

		if requestsRemainder > 0 && i == lambdas-1 {
			requests += requestsRemainder
		}

		c := t.config
		args := []string{
			"-u",
			fmt.Sprintf("%s", c.URL),
			"-c",
			fmt.Sprintf("%s", strconv.Itoa(int(concurrency))),
			"-n",
			fmt.Sprintf("%s", strconv.Itoa(int(requests))),
			"-s",
			fmt.Sprintf("%s", sqsURL),
			"-q",
			fmt.Sprintf("%s", c.Regions[0]),
			"-t",
			fmt.Sprintf("%s", c.RequestTimeout.String()),
			"-f",
			fmt.Sprintf("%s", reportingFrequency(lambdas).String()),
			"-r",
			fmt.Sprintf("%s", region),
			"-m",
			fmt.Sprintf("%s", c.Method),
			"-b",
			fmt.Sprintf("%s", c.Body),
		}

		for _, v := range t.config.Headers {
			args = append(args, "-H", fmt.Sprintf("%s", v))
		}

		invokeargs := invokeArgs{
			File: "./goad-lambda",
			Args: args,
		}

		config := aws.NewConfig().WithRegion(region)
		go t.invokeLambda(config, invokeargs)
	}
}

func (t *Test) invokeLambda(awsConfig *aws.Config, args invokeArgs) {
	svc := lambda.New(session.New(), awsConfig)

	j, _ := json.Marshal(args)

	svc.InvokeAsync(&lambda.InvokeAsyncInput{
		FunctionName: aws.String("goad:" + version.LambdaVersion()),
		InvokeArgs:   bytes.NewReader(j),
	})
}

func numberOfLambdas(concurrency uint, numRegions int) int {
	if numRegions > int(concurrency) {
		return int(concurrency)
	}
	if concurrency/200 > 350 { // > 70.000
		return 500
	} else if concurrency/100 > 100 { // 10.000 <> 70.000
		return 300
	} else if concurrency/10 > 100 { // 1.000 <> 10.000
		return 100
	}
	if int(concurrency) < 10*numRegions {
		return numRegions
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
	concurrencyLimit := 25000 * uint(len(c.Regions))
	if c.Concurrency < 1 || c.Concurrency > concurrencyLimit {
		return fmt.Errorf("Invalid concurrency (use 1 - %d)", concurrencyLimit)
	}
	if c.TotalRequests < 1 || c.TotalRequests > 2000000 {
		return errors.New("Invalid total requests (use 1 - 2000000)")
	}
	if c.RequestTimeout.Nanoseconds() < nano || c.RequestTimeout.Nanoseconds() > nano*100 {
		return errors.New("Invalid timeout (1s - 100s)")
	}
	for _, region := range c.Regions {
		supportedRegionFound := false
		for _, supported := range supportedRegions {
			if region == supported {
				supportedRegionFound = true
			}
		}
		if !supportedRegionFound {
			return fmt.Errorf("Unsupported region: %s. Supported regions are: %s.", region, strings.Join(supportedRegions, ", "))
		}
	}
	for _, v := range c.Headers {
		header := strings.Split(v, ":")
		if len(header) < 2 {
			return fmt.Errorf("Header %s not valid. Make sure your header is of the form \"Header: value\"", v)
		}
	}
	return nil
}
