package sqs

import (
	"fmt"
	"testing"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/goadapp/goad/api"
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
	assert := assert.New(t)
	// This test just verifies the json api.
	result := api.RunnerResult{
		RequestCount:     299,
		TimedOut:         234,
		ConnectionErrors: 256,
		AveTimeToFirst:   9999,
		BytesRead:        2136,
		// Statuses:         new(map[string]int),
		AveTimeForReq: 12345,
		// AveReqPerSec:         6789,
		// AveKBytesPerSec:      6789,
		Slowest:    4567,
		Fastest:    4567,
		Region:     "eu-west",
		RunnerID:   0,
		FatalError: "sorry",
	}
	str, jsonerr := jsonFromResult(result)
	if jsonerr != nil {
		fmt.Println(jsonerr)
		return
	}
	json, err := resultFromJSON(str)
	if err != nil {
		t.Fatal(err)
	}
	assert.EqualValues(result, json, "Should serialize and deserialize without error and loosing information")
}
