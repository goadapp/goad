package infrastructure

import (
	"fmt"
	"math"
	"strconv"
	"time"

	"github.com/goadapp/goad/goad/types"
	"github.com/goadapp/goad/result"
)

const DefaultRunnerAsset = "data/lambda.zip"

type Infrastructure interface {
	Setup() (teardown func(), err error)
	Run(args InvokeArgs)
	GetQueueURL() string
	Receive(chan *result.LambdaResults)
	GetSettings() *types.TestConfig
}

type InvokeArgs struct {
	File string   `json:"file"`
	Args []string `json:"args"`
}

func InvokeLambdas(inf Infrastructure) {
	t := inf.GetSettings()
	currentID := 0
	for i := 0; i < t.Lambdas; i++ {
		region := t.Regions[i%len(t.Regions)]
		requests, requestsRemainder := divide(t.Requests, t.Lambdas)
		concurrency, _ := divide(t.Concurrency, t.Lambdas)
		execTimeout := t.Timelimit

		if requestsRemainder > 0 && i == t.Lambdas-1 {
			requests += requestsRemainder
		}

		args := []string{
			fmt.Sprintf("--concurrency=%s", strconv.Itoa(int(concurrency))),
			fmt.Sprintf("--requests=%s", strconv.Itoa(int(requests))),
			fmt.Sprintf("--execution-time=%s", strconv.Itoa(int(execTimeout))),
			fmt.Sprintf("--sqsurl=%s", inf.GetQueueURL()),
			fmt.Sprintf("--queue-region=%s", t.Regions[0]),
			fmt.Sprintf("--client-timeout=%s", time.Duration(t.Timeout)*time.Second),
			fmt.Sprintf("--frequency=%s", reportingFrequency(t.Lambdas).String()),
			fmt.Sprintf("--aws-region=%s", region),
			fmt.Sprintf("--method=%s", t.Method),
			fmt.Sprintf("--runner-id=%d", currentID),
			fmt.Sprintf("--body=%s", t.Body),
		}
		currentID++
		for _, v := range t.Headers {
			args = append(args, fmt.Sprintf("--header=%s", v))
		}
		args = append(args, fmt.Sprintf("%s", t.URL))

		invokeargs := InvokeArgs{
			File: "./goad-lambda",
			Args: args,
		}

		go inf.Run(invokeargs)
	}
}

func Aggregate(i Infrastructure) chan *result.LambdaResults {
	results := make(chan *result.LambdaResults)
	go i.Receive(results)
	return results
}

func divide(dividend int, divisor int) (quotient, remainder int) {
	return dividend / divisor, dividend % divisor
}

func reportingFrequency(numberOfLambdas int) time.Duration {
	return time.Duration((math.Log2(float64(numberOfLambdas)) + 1)) * time.Second
}
