package sqsadaptor

import (
	"encoding/json"
	"fmt"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/sqs"
)

// Result test result
type Result struct {
	Time      string `json:"time"`
	Host      string `json:"host"`
	Type      string `json:"type"`
	RequestID string `json:"requestID"`
	Status    int    `json:"status"`
	Elapsed   int    `json:"elapsed"`
	Bytes     int    `json:"bytes"`
	State     string `json:"state"`
}

// SQSAdaptor is used to send messages to the queue
type SQSAdaptor struct {
	Client   *sqs.SQS
	QueueURL string
}

// New returns a new sqs interface object
func New(queueURL string) SQSAdaptor {
	return SQSAdaptor{getClient(), queueURL}
}

func getClient() *sqs.SQS {
	client := sqs.New(session.New())
	return client
}

func messageJSON(result Result) (string, error) {
	data, jsonerr := json.Marshal(result)
	if jsonerr != nil {
		return "", jsonerr
	}
	return string(data), nil
}

// SendResult adds a result to the queue
func (adaptor SQSAdaptor) SendResult(result Result) {
	str, jsonerr := messageJSON(result)
	if jsonerr != nil {
		fmt.Println(jsonerr)
		return
	}
	params := &sqs.SendMessageInput{
		MessageBody: aws.String(str),
		QueueUrl:    aws.String(adaptor.QueueURL),
	}
	_, err := adaptor.Client.SendMessage(params)

	if err != nil {
		fmt.Println(err.Error())
		return
	}
}
