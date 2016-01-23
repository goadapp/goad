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

// DummyAdaptor is used to send messages to the screen for testing
type DummyAdaptor struct {
	QueueURL string
}

// ResultAdaptor defines the methods needed to return the result
type ResultAdaptor interface {
	SendResult(result Result)
}

// NewSQSAdaptor returns a new sqs adator object
func NewSQSAdaptor(queueURL string) SQSAdaptor {
	return SQSAdaptor{getClient(), queueURL}
}

// NewDummyAdaptor returns a new sqs adator object
func NewDummyAdaptor(queueURL string) DummyAdaptor {
	return DummyAdaptor{queueURL}
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

// SendResult prints the result
func (adaptor DummyAdaptor) SendResult(result Result) {
	str, jsonerr := messageJSON(result)
	if jsonerr != nil {
		fmt.Println(jsonerr)
		return
	}
	fmt.Println(str)
}
