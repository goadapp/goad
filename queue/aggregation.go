package queue

import (
	"time"

	"github.com/gophergala2016/goad/Godeps/_workspace/src/github.com/aws/aws-sdk-go/aws"
)

// AggData type
type AggData struct {
	TotalReqs            int            `json:"total-reqs"`
	TotalTimedOut        int            `json:"total-timed-out"`
	TotalConnectionError int            `json:"total-conn-error"`
	AveTimeToFirst       int64          `json:"ave-time-to-first"`
	TotBytesRead         int            `json:"tot-bytes-read"`
	Statuses             map[string]int `json:"statuses"`
	AveTimeForReq        int64          `json:"ave-time-for-req"`
	AveReqPerSec         float32        `json:"ave-req-per-sec"`
	AveKBytesPerSec      float32        `json:"ave-kbytes-per-sec"`
	Slowest              int64          `json:"slowest"`
	Fastest              int64          `json:"fastest"`
	Region               string         `json:"region"`
	FatalError           string         `json:"fatal-error"`
}

// RegionsAggData type
type RegionsAggData struct {
	Regions               map[string]AggData
	TotalExpectedRequests uint
}

func (d *RegionsAggData) allRequestsReceived() bool {
	var requests uint

	for _, region := range d.Regions {
		requests += uint(region.TotalReqs)
	}

	return requests == d.TotalExpectedRequests
}

func addResult(data *AggData, result *AggData) {
	initialDataTot := data.TotalReqs
	initialDataTot64 := int64(data.TotalReqs)
	data.TotalReqs += result.TotalReqs
	data.TotalTimedOut += result.TotalTimedOut
	dataTot64 := int64(data.TotalReqs)
	resultTot64 := int64(result.TotalReqs)
	if dataTot64 > 0 {
		data.AveTimeToFirst = (data.AveTimeToFirst*initialDataTot64 + result.AveTimeToFirst*resultTot64) / dataTot64
		data.AveTimeForReq = (data.AveTimeForReq*initialDataTot64 + result.AveTimeForReq*resultTot64) / dataTot64
		data.AveReqPerSec = (data.AveReqPerSec*float32(initialDataTot) + result.AveReqPerSec*float32(result.TotalReqs)) / float32(data.TotalReqs)
	}
	data.TotBytesRead += result.TotBytesRead

	for key, value := range result.Statuses {
		data.Statuses[key] += value
	}

	if result.Slowest > data.Slowest {
		data.Slowest = result.Slowest
	}
	if data.Fastest == 0 || result.Fastest < data.Fastest {
		data.Fastest = result.Fastest
	}
}

// Aggregate listens for results and sends totals, closing the channel when done
func Aggregate(awsConfig *aws.Config, queueURL string, totalExpectedRequests uint) chan RegionsAggData {
	results := make(chan RegionsAggData)
	go aggregate(results, awsConfig, queueURL, totalExpectedRequests)
	return results
}

func aggregate(results chan RegionsAggData, awsConfig *aws.Config, queueURL string, totalExpectedRequests uint) {
	defer close(results)
	data := RegionsAggData{make(map[string]AggData), totalExpectedRequests}

	adaptor := NewSQSAdaptor(awsConfig, queueURL)
	timeoutStart := time.Now()
	for {
		result := adaptor.Receive()
		if result != nil {
			regionData, ok := data.Regions[result.Region]
			if !ok {
				regionData.Statuses = make(map[string]int)
				regionData.Region = result.Region
			}
			addResult(&regionData, result)
			data.Regions[result.Region] = regionData
			results <- data
			if data.allRequestsReceived() {
				break
			}
			timeoutStart = time.Now()
		} else {
			waited := time.Since(timeoutStart)
			if waited.Seconds() > 20 {
				break
			}
		}
	}
}
