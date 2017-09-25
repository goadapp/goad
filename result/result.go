package result

import (
	"math"
	"sort"
	"time"

	"github.com/goadapp/goad/api"
	"github.com/goadapp/goad/goad/util"
)

// AggData type
type AggData struct {
	TotalReqs            int
	TotalTimedOut        int
	TotalConnectionError int
	AveTimeToFirst       int64
	TotBytesRead         int
	Statuses             map[string]int
	AveTimeForReq        int64
	AveReqPerSec         float64
	TimeDelta            time.Duration
	AveKBytesPerSec      float64
	Slowest              int64
	Fastest              int64
	Region               string
	FatalError           string
	Finished             bool
}

// LambdaResults type
type LambdaResults struct {
	Lambdas []AggData
}

// Regions the LambdaResults were collected from
func (r *LambdaResults) Regions() []string {
	regions := make([]string, 0)
	for _, lambda := range r.Lambdas {
		if lambda.Region != "" {
			regions = append(regions, lambda.Region)
		}
	}
	regions = util.RemoveDuplicates(regions)
	sort.Strings(regions)
	return regions
}

// RegionsData aggregates the individual lambda functions results per region
func (r *LambdaResults) RegionsData() map[string]AggData {
	regionsMap := make(map[string]AggData)
	for _, region := range r.Regions() {
		regionsMap[region] = sumAggData(r.ResultsForRegion(region))
	}
	return regionsMap
}

//SumAllLambdas aggregates results of all Lambda functions
func (r *LambdaResults) SumAllLambdas() AggData {
	return sumAggData(r.Lambdas)
}

//ResultsForRegion return the sum of results for a given regions
func (r *LambdaResults) ResultsForRegion(region string) []AggData {
	lambdasOfRegion := make([]AggData, 0)
	for _, lambda := range r.Lambdas {
		if lambda.Region == region {
			lambdasOfRegion = append(lambdasOfRegion, lambda)
		}
	}
	return lambdasOfRegion
}

func SetupRegionsAggData(lambdaCount int) *LambdaResults {
	lambdaResults := &LambdaResults{
		Lambdas: make([]AggData, lambdaCount),
	}
	for i := 0; i < lambdaCount; i++ {
		lambdaResults.Lambdas[i].Statuses = make(map[string]int)
	}
	return lambdaResults
}

func sumAggData(dataArray []AggData) AggData {
	sum := AggData{
		Fastest:  math.MaxInt64,
		Statuses: make(map[string]int),
		Finished: true,
	}
	for _, lambda := range dataArray {
		sum.AveKBytesPerSec += lambda.AveKBytesPerSec
		sum.AveReqPerSec += lambda.AveReqPerSec
		sum.AveTimeForReq += lambda.AveTimeForReq
		sum.AveTimeToFirst += lambda.AveTimeToFirst
		if lambda.Fastest < sum.Fastest {
			sum.Fastest = lambda.Fastest
		}
		sum.FatalError += lambda.FatalError
		if !lambda.Finished {
			sum.Finished = false
		}
		sum.Region = lambda.Region
		if lambda.Slowest > sum.Slowest {
			sum.Slowest = lambda.Slowest
		}
		for key := range lambda.Statuses {
			sum.Statuses[key] += lambda.Statuses[key]
		}
		sum.TimeDelta += lambda.TimeDelta
		sum.TotalConnectionError += lambda.TotalConnectionError
		sum.TotalReqs += lambda.TotalReqs
		sum.TotalTimedOut += lambda.TotalTimedOut
		sum.TotBytesRead += lambda.TotBytesRead
	}
	sum.AveTimeForReq = sum.AveTimeForReq / int64(len(dataArray))
	sum.AveTimeToFirst = sum.AveTimeToFirst / int64(len(dataArray))
	return sum
}

func (r *LambdaResults) AllLambdasFinished() bool {
	for _, lambda := range r.Lambdas {
		if !lambda.Finished {
			return false
		}
	}
	return true
}

func AddResult(data *AggData, result *api.RunnerResult) {
	initCountOk := int64(data.TotalReqs - data.TotalTimedOut - data.TotalConnectionError)
	addCountOk := int64(result.RequestCount - result.TimedOut - result.ConnectionErrors)
	totalCountOk := initCountOk + addCountOk

	data.TotalReqs += result.RequestCount
	data.TotalTimedOut += result.TimedOut
	data.TotalConnectionError += result.ConnectionErrors
	data.TotBytesRead += result.BytesRead
	data.TimeDelta += result.TimeDelta

	if totalCountOk > 0 {
		data.AveTimeToFirst = addToTotalAverage(data.AveTimeToFirst, initCountOk, result.AveTimeToFirst, addCountOk)
		data.AveTimeForReq = addToTotalAverage(data.AveTimeForReq, initCountOk, result.AveTimeForReq, addCountOk)
		data.AveKBytesPerSec = float64(data.TotBytesRead) / float64(data.TimeDelta.Seconds())
		data.AveReqPerSec = float64(data.TotalReqs) / float64(data.TimeDelta.Seconds())
	}

	for key, value := range result.Statuses {
		data.Statuses[key] += value
	}

	if result.Slowest > data.Slowest {
		data.Slowest = result.Slowest
	}
	if result.Fastest > 0 && (data.Fastest == 0 || result.Fastest < data.Fastest) {
		data.Fastest = result.Fastest
	}
	data.Finished = result.Finished
	data.Region = result.Region
}

func addToTotalAverage(currentAvg, currentCount, addAvg, addCount int64) int64 {
	return ((currentAvg * currentCount) + (addAvg * addCount)) / (currentCount + addCount)
}

func addToTotalAverageFloat(currentAvg, currentCount, addAvg, addCount float64) float64 {
	return ((currentAvg * currentCount) + (addAvg * addCount)) / (currentCount + addCount)
}
