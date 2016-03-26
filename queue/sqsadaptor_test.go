package queue

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
)

func init() {
}

func TestAdaptorConstruction(t *testing.T) {
	testsqs := NewSQSAdaptor("testqueue")
	assert.Equal(t, testsqs.QueueURL, "testqueue")
}

func TestJSON(t *testing.T) {
	result := AggData{
		299,
		234,
		int64(9999),
		2136,
		make(map[string]int),
		int64(12345),
		6789,
		int64(4567),
		int64(4567),
		"eu-west",
	}
	str, jsonerr := jsonFromResult(result)
	if jsonerr != nil {
		fmt.Println(jsonerr)
		return
	}
	assert.Equal(t, str, "{\"total-reqs\":299,\"total-timed-out\":234,\"ave-time-to-first\":9999,\"tot-bytes-read\":2136,\"statuses\":{},\"ave-time-for-req\":12345,\"ave-req-per-sec\":6789,\"slowest\":4567,\"fastest\":4567,\"region\":\"eu-west\"}")
}
