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

	t.invokeLambda(awsConfig, infra.QueueURL())

	results := make(chan sqsadaptor.RegionsAggData)

	go func() {
		sqsadaptor.Aggregate(results, awsConfig, infra.QueueURL(), t.config.TotalRequests)
	}()
	for result := range results {
		fmt.Printf("%#v\n", result)
	}
	return nil
}

func (t *Test) invokeLambda(awsConfig *aws.Config, sqsURL string) {
	svc := lambda.New(session.New(), awsConfig)

	resp, err := svc.InvokeAsync(&lambda.InvokeAsyncInput{
		FunctionName: aws.String("goad"),
		InvokeArgs:   strings.NewReader(`{"cmd":"` + t.config.cmd(sqsURL) + `"}`),
	})
	fmt.Println(resp, err)
}
