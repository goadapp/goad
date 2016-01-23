package goad

import (
	"log"
	"time"

	"github.com/aws/aws-sdk-go/aws"
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

	results := make(chan sqsadaptor.RegionsAggData)
	sqsadaptor.Aggregate(results, infra.QueueURL(), t.config.TotalRequests)

	// TODO: add result channel
	return nil
}
