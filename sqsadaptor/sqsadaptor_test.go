package sqsadaptor

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
	result := Result{
		"2016-01-01 10:00:00",
		"example.com",
		"Fetch",
		200,
		int64(12345), // elapsed, first byte
		int64(6789),  // elapsed, last byte
		int64(4567),  // elapsed, total
		4711,         //bytes
		"Success",
		"aws-lambda-4711", // AWS lamba instance

	}
	str, jsonerr := jsonFromResult(result)
	if jsonerr != nil {
		fmt.Println("JSON error")
		return
	}
	assert.Equal(t, str, "{\"time\":\"2016-01-01 10:00:00\",\"host\":\"example.com\",\"type\":\"Fetch\",\"status\":200,\"elapsed-first-byte\":12345,\"elapsed-last-byte\":6789,\"elapsed\":4567,\"bytes\":4711,\"state\":\"Success\",\"instance\":\"aws-lambda-4711\"}")
}
