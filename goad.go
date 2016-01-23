package goad

import (
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/lambda"
	"github.com/gophergala2016/goad/infrastructure"
	"github.com/gophergala2016/goad/sqsadaptor"
)

type Result struct{}

type TestConfig struct {
	URL            string
	Concurrency    uint
	TotalRequests  uint
	RequestTimeout time.Duration
	Region         string
}

func (c *TestConfig) cmd(sqsURL string) string {
	return fmt.Sprintf("./goad-lambda %s %d %d %s %s", c.URL, c.Concurrency, c.TotalRequests, sqsURL, c.Region)
}

type Test struct {
	config *TestConfig
}

func NewTest(config *TestConfig) *Test {
	return &Test{config}
}

func (t *Test) Start() <-chan Result {
	awsConfig := aws.NewConfig().WithRegion(t.config.Region)
	infra, err := infrastructure.New(awsConfig)
	if err != nil {
		log.Fatal(err)
	}
	defer infra.Clean()

	t.invokeLambda(infra.QueueURL())

	results := make(chan sqsadaptor.RegionsAggData)
	sqsadaptor.Aggregate(results, infra.QueueURL(), t.config.TotalRequests)

	for result := range results {
		fmt.Println(result)
	}
	return nil
}

func (t *Test) invokeLambda(sqsURL string) {
	svc := lambda.New(session.New())

	resp, err := svc.InvokeAsync(&lambda.InvokeAsyncInput{
		FunctionName: aws.String("goad"),
		InvokeArgs:   strings.NewReader(`{"cmd":"` + t.config.cmd(sqsURL) + `"}`),
	})
	fmt.Println(resp, err)
}
