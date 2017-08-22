package goad

import (
	"github.com/goadapp/goad/goad/types"
	"github.com/goadapp/goad/infrastructure"
	"github.com/goadapp/goad/infrastructure/aws"
	"github.com/goadapp/goad/infrastructure/docker"
	"github.com/goadapp/goad/result"
)

// Start a test
func Start(t *types.TestConfig) (<-chan *result.LambdaResults, func()) {

	var infra infrastructure.Infrastructure
	if t.RunDocker {
		infra = dockerinfra.New(t)
	} else {
		infra = awsinfra.New(t)
	}
	teardown, err := infra.Setup()
	HandleErr(err)
	t.Lambdas = numberOfLambdas(t.Concurrency, len(t.Regions))
	infrastructure.InvokeLambdas(infra)

	results := make(chan *result.LambdaResults)

	go func() {
		for result := range infrastructure.Aggregate(infra) {
			results <- result
		}
		close(results)
	}()

	return results, teardown
}

func HandleErr(err error) {
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
