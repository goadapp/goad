package sqsadaptor

import "time"

// AggData type
type AggData struct {
	TotalReqs      int
	TotalTimedOut  int
	AveTimeToFirst int
	TotBytesRead   int
	Statuses       map[int]int
	AveTimeForReq  int64
	AveReqPerSec   int
	Slowest        int64
	Fastest        int64
}

// RegionsAggData type
type RegionsAggData struct {
	Regions map[string]AggData
}

// Aggregate begins listening for results
func Aggregate(outChan chan RegionsAggData, queueURL string, totalExpectedRequests int) {
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
