package goad

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"math"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/lambda"
	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/client"
	"github.com/goadapp/goad/infrastructure"
	"github.com/goadapp/goad/queue"
	"github.com/goadapp/goad/version"
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
	"ap-northeast-2",
	"ap-southeast-1",
	"ap-southeast-2",
}

// Test type
type Test struct {
	Config  *TestConfig
	infra   *infrastructure.Infrastructure
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
func (t *Test) Start() <-chan queue.RegionsAggData {
	awsConfig := aws.NewConfig().WithRegion(t.Config.Regions[0])

	infra, err := infrastructure.New(t.Config.Regions, awsConfig)
	if err != nil {
		log.Fatal(err)
	}

	t.infra = infra
	t.lambdas = numberOfLambdas(t.Config.Concurrency, len(t.Config.Regions))
	t.invokeLambdas(awsConfig, infra.QueueURL())

	results := make(chan queue.RegionsAggData)

	go func() {
		for result := range queue.Aggregate(awsConfig, infra.QueueURL(), t.Config.Requests, t.lambdas) {
			results <- result
		}
		close(results)
	}()

	return results
}

func (t *Test) invokeLambdas(awsConfig *aws.Config, sqsURL string) {
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
			fmt.Sprintf("--sqsurl=%s", sqsURL),
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

		invokeargs := invokeArgs{
			File: "./goad-lambda",
			Args: args,
		}

		config := aws.NewConfig().WithRegion(region)

		if t.Config.RunDocker {
			go runAsDockerContainer(config, invokeargs)
		} else {
			go invokeLambda(config, invokeargs)
		}
	}
}

func handleErr(err error) {
	if err != nil {
		panic(err)
	}
}

func PullDockerImage() {
	ctx := context.Background()
	cli, err := client.NewEnvClient()
	handleErr(err)

	// Pull the latest lambci/lambda image from dockerhub.
	out, err := cli.ImagePull(ctx, "lambci/lambda", types.ImagePullOptions{})
	handleErr(err)
	defer out.Close()
	io.Copy(os.Stdout, out)
}

func runAsDockerContainer(config *aws.Config, args invokeArgs) {
	ctx := context.Background()
	cli, err := client.NewEnvClient()
	handleErr(err)

	session := session.New()
	conf := session.ClientConfig("lambda", config)
	credValue, err := conf.Config.Credentials.Get()
	handleErr(err)

	accessKeyID := fmt.Sprintf("AWS_ACCESS_KEY_ID=%s", credValue.AccessKeyID)
	secretAccessKey := fmt.Sprintf("AWS_SECRET_ACCESS_KEY=%s", credValue.SecretAccessKey)
	sessionToken := fmt.Sprintf("AWS_SESSION_TOKEN=%s", credValue.SessionToken)

	// Create container to execute lambda
	resp, err := cli.ContainerCreate(ctx, &container.Config{
		Image: "lambci/lambda",
		Cmd:   append([]string{"index.handler"}, toJSONString(args)),
		Volumes: map[string]struct{}{
			"/var/task": struct{}{},
		},
		Env: []string{accessKeyID, secretAccessKey, sessionToken},
	}, &container.HostConfig{
		AutoRemove: true,
		Binds:      []string{os.ExpandEnv("${PWD}/data/lambda:/var/task:ro")},
	}, nil, "")
	handleErr(err)

	// run container
	err = cli.ContainerStart(ctx, resp.ID, types.ContainerStartOptions{})
	handleErr(err)

	// log container output
	// out, err = cli.ContainerLogs(ctx, resp.ID, types.ContainerLogsOptions{
	// 	Follow:     true,
	// 	ShowStderr: true,
	// 	ShowStdout: true,
	// })
	// handleErr(err)
	// defer out.Close()
	// io.Copy(os.Stdout, out)
}

func toJSONString(args interface{}) string {
	b, err := json.Marshal(args)
	handleErr(err)
	return string(b[:])
}
func toJSONReadSeeker(args interface{}) io.ReadSeeker {
	j, err := json.Marshal(args)
	handleErr(err)
	return bytes.NewReader(j)
}

func invokeLambda(awsConfig *aws.Config, args invokeArgs) {
	svc := lambda.New(session.New(), awsConfig)

	svc.InvokeAsync(&lambda.InvokeAsyncInput{
		FunctionName: aws.String("goad:" + version.LambdaVersion()),
		InvokeArgs:   toJSONReadSeeker(args),
	})
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

func (t *Test) Clean() {
	if t.infra != nil {
		t.infra.Clean()
	}
}
