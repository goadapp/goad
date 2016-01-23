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
		"928429348",
		200,
		238947,
		2398,
		"Finished",
	}
	str, jsonerr := messageJSON(result)
	if jsonerr != nil {
		fmt.Println("JSON error")
		return
	}
	assert.Equal(t, str, "{\"time\":\"2016-01-01 10:00:00\",\"host\":\"example.com\",\"type\":\"Fetch\",\"requestID\":\"928429348\",\"status\":200,\"elapsed\":238947,\"bytes\":2398,\"state\":\"Finished\"}")
}
