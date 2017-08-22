package sqsadapter

import (
	"testing"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/stretchr/testify/assert"
)

func TestAdapterConstruction(t *testing.T) {
	config := aws.NewConfig().WithRegion("somewhere")
	testsqs := New(config, "testqueue")
	assert.Equal(t, testsqs.QueueURL, "testqueue")
}
