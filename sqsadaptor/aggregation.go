package sqsadaptor

import "time"

// AggData type
type AggData struct {
	TotalReqs      int         `json:"total-reqs"`
	TotalTimedOut  int         `json:"total-timed-out"`
	AveTimeToFirst int         `json:"ave-time-to-first"`
	TotBytesRead   int         `json:"tot-bytes-read"`
	Statuses       map[int]int `json:"statuses"`
	AveTimeForReq  int64       `json:"ave-time-for-req"`
	AveReqPerSec   int         `json:"ave-req-per-sec"`
	Slowest        int64       `json:"slowest"`
	Fastest        int64       `json:"fastest"`
}

// RegionsAggData type
type RegionsAggData struct {
	Regions map[string]AggData
}

// Aggregate listens for results and sends totals
func Aggregate(outChan chan RegionsAggData, queueURL string, totalExpectedRequests uint) {
	defer close(outChan)
	data := RegionsAggData{make(map[string]AggData)}

	adaptor := NewSQSAdaptor(queueURL)
	timeoutStart := time.Now()
	for {
		result := adaptor.Receive()
		if result != nil {

			outChan <- data
			timeoutStart = time.Now()
		} else {
			waited := time.Since(timeoutStart)
			if waited.Seconds() > 10 {
				break
			}
		}
	}
}
