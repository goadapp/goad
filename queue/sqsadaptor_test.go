package queue

import (
	"fmt"
	"testing"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/stretchr/testify/assert"
)

func init() {
}

func TestAdapterConstruction(t *testing.T) {
	config := aws.NewConfig().WithRegion("somewhere")
	testsqs := NewSQSAdapter(config, "testqueue")
	assert.Equal(t, testsqs.QueueURL, "testqueue")
}

func TestJSON(t *testing.T) {
	// This test just verifies the json api.
	result := AggData{
		TotalReqs:            299,
		TotalTimedOut:        234,
		TotalConnectionError: 256,
		AveTimeToFirst:       9999,
		TotBytesRead:         2136,
		Statuses:             make(map[string]int),
		AveTimeForReq:        12345,
		AveReqPerSec:         6789,
		AveKBytesPerSec:      6789,
		Slowest:              4567,
		Fastest:              4567,
		Region:               "eu-west",
		FatalError:           "sorry",
	}
	str, jsonerr := jsonFromResult(result)
	if jsonerr != nil {
		fmt.Println(jsonerr)
		return
	}
	assert.Equal(t, str, "{\"total-reqs\":299,\"total-timed-out\":234,\"total-conn-error\":256,\"ave-time-to-first\":9999,\"tot-bytes-read\":2136,\"statuses\":{},\"ave-time-for-req\":12345,\"ave-req-per-sec\":6789,\"ave-kbytes-per-sec\":6789,\"slowest\":4567,\"fastest\":4567,\"region\":\"eu-west\",\"fatal-error\":\"sorry\",\"finished\":false,\"finished-lambdas\":0}")
}
