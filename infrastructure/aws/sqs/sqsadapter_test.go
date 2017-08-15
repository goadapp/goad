package sqs

import (
	"testing"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/stretchr/testify/assert"
)

func TestAdapterConstruction(t *testing.T) {
	config := aws.NewConfig().WithRegion("somewhere")
	testsqs := NewSQSAdapter(config, "testqueue")
	assert.Equal(t, testsqs.QueueURL, "testqueue")
}
