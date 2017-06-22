package goad

import (
	"errors"
	"fmt"
	"math"
	"strconv"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/goadapp/goad/infrastructure"
	"github.com/goadapp/goad/infrastructure/aws"
	"github.com/goadapp/goad/infrastructure/docker"
	"github.com/goadapp/goad/queue"
)

// TestConfig type
type TestConfig struct {
	URL         string
	Concurrency int
	Requests    int
	Timelimit   int
	Timeout     int
	Regions     []string
	Method      string
	Body        string
	Headers     []string
	Output      string
	Settings    string
	RunDocker   bool
}

const nano = 1000000000

var supportedRegions = []string{
	"us-east-1",      // N. Virginia
	"us-west-1",      // N.California
	"us-west-2",      // Oregon
	"eu-west-1",      // Ireland
	"eu-central-1",   // Frankfurt
	"ap-northeast-1", // Sydney
	"ap-northeast-2", // Seoul
	"ap-southeast-1", // Singapore
	"ap-southeast-2", // Tokio
	"sa-east-1",      // Sao Paulo
}

// Test type
type Test struct {
	Config  *TestConfig
	infra   infrastructure.Infrastructure
	lambdas int
}

// NewTest returns a configured Test
func NewTest(config *TestConfig) (*Test, error) {
	err := config.check()
	if err != nil {
		return nil, err
	}
	return &Test{Config: config, infra: nil}, nil
}

// Start a test
func (t *Test) Start() (<-chan queue.RegionsAggData, func()) {
	awsConfig := aws.NewConfig().WithRegion(t.Config.Regions[0])

	if t.Config.RunDocker {
		t.infra = dockerinfra.NewDockerInfrastructure()
	} else {
		t.infra = awsinfra.New(t.Config.Regions, awsConfig)
	}
	teardown, err := t.infra.Setup()
	handleErr(err)
	t.lambdas = numberOfLambdas(t.Config.Concurrency, len(t.Config.Regions))
	t.invokeLambdas(awsConfig, t.infra.GetQueueURL())

	results := make(chan queue.RegionsAggData)

	go func() {
		for result := range queue.Aggregate(awsConfig, t.infra.GetQueueURL(), t.Config.Requests, t.lambdas) {
			results <- result
		}
		close(results)
	}()

	return results, teardown
}

func (t *Test) invokeLambdas(awsConfig *aws.Config, queueURL string) {
	for i := 0; i < t.lambdas; i++ {
		region := t.Config.Regions[i%len(t.Config.Regions)]
		requests, requestsRemainder := divide(t.Config.Requests, t.lambdas)
		concurrency, _ := divide(t.Config.Concurrency, t.lambdas)
		execTimeout := t.Config.Timelimit

		if requestsRemainder > 0 && i == t.lambdas-1 {
			requests += requestsRemainder
		}

		c := t.Config
		args := []string{
			fmt.Sprintf("--concurrency=%s", strconv.Itoa(int(concurrency))),
			fmt.Sprintf("--requests=%s", strconv.Itoa(int(requests))),
			fmt.Sprintf("--execution-time=%s", strconv.Itoa(int(execTimeout))),
			fmt.Sprintf("--sqsurl=%s", queueURL),
			fmt.Sprintf("--queue-region=%s", c.Regions[0]),
			fmt.Sprintf("--client-timeout=%s", time.Duration(c.Timeout)*time.Second),
			fmt.Sprintf("--frequency=%s", reportingFrequency(t.lambdas).String()),
			fmt.Sprintf("--aws-region=%s", region),
			fmt.Sprintf("--method=%s", c.Method),
			fmt.Sprintf("--body=%s", c.Body),
		}
		for _, v := range t.Config.Headers {
			args = append(args, fmt.Sprintf("--header=%s", v))
		}
		args = append(args, fmt.Sprintf("%s", c.URL))

		invokeargs := infrastructure.InvokeArgs{
			File: "./goad-lambda",
			Args: args,
		}

		go t.infra.Run(invokeargs)
	}
}

func handleErr(err error) {
	if err != nil {
		panic(err)
	}
}

func numberOfLambdas(concurrency int, numRegions int) int {
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

func divide(dividend int, divisor int) (quotient, remainder int) {
	return dividend / divisor, dividend % divisor
}

func reportingFrequency(numberOfLambdas int) time.Duration {
	return time.Duration((math.Log2(float64(numberOfLambdas)) + 1)) * time.Second
}

func (c TestConfig) check() error {
	concurrencyLimit := 25000 * len(c.Regions)
	if c.Concurrency < 1 || c.Concurrency > concurrencyLimit {
		return fmt.Errorf("Invalid concurrency (use 1 - %d)", concurrencyLimit)
	}
	if (c.Requests < 1 && c.Timelimit <= 0) || c.Requests > 2000000 {
		return errors.New("Invalid total requests (use 1 - 2000000)")
	}
	if c.Timelimit > 3600 {
		return errors.New("Invalid maximum execution time in seconds (use 0 - 3600)")
	}
	if c.Timeout < 1 || c.Timeout > 100 {
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
