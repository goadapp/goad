package queue

import (
	// "fmt"
	"testing"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/stretchr/testify/assert"
)

func init() {
}

func TestAdaptorConstruction(t *testing.T) {
	config := aws.NewConfig().WithRegion("somewhere")
	testsqs := NewSQSAdaptor(config, "testqueue")
	assert.Equal(t, testsqs.QueueURL, "testqueue")
}

func TestJSON(t *testing.T) {
	// result := AggData{
	// 	299,
	// 	234,
	// 	256,
	// 	int64(9999),
	// 	2136,
	// 	make(map[string]int),
	// 	int64(12345),
	// 	float32(6789),
	// 	float32(6789),
	// 	int64(4567),
	// 	int64(4567),
	// 	"eu-west",
	// 	"sorry",
	// }
	// str, jsonerr := jsonFromResult(result)
	// if jsonerr != nil {
	// 	fmt.Println(jsonerr)
	// 	return
	// }
	// assert.Equal(t, str, "{\"total-reqs\":299,\"total-timed-out\":234,\"total-conn-error\":256,\"ave-time-to-first\":9999,\"tot-bytes-read\":2136,\"statuses\":{},\"ave-time-for-req\":12345,\"ave-req-per-sec\":6789,\"ave-kbytes-per-sec\":6789,\"slowest\":4567,\"fastest\":4567,\"region\":\"eu-west\",\"fatal-error\":\"sorry\"}")
}
