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
	Time             string `json:"time"`
	Host             string `json:"host"`
	Type             string `json:"type"`
	Status           int    `json:"status"`
	ElapsedFirstByte int64  `json:"elapsed-first-byte"`
	ElapsedLastByte  int64  `json:"elapsed-last-byte"`
	Elapsed          int64  `json:"elapsed"`
	Bytes            int    `json:"bytes"`
	State            string `json:"state"`
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

// Receive a result, or timeout in 1 second
func (adaptor SQSAdaptor) Receive() {
	params := &sqs.ReceiveMessageInput{
		QueueUrl:            aws.String(adaptor.QueueURL),
		MaxNumberOfMessages: aws.Int64(1),
		VisibilityTimeout:   aws.Int64(1),
		WaitTimeSeconds:     aws.Int64(1),
	}
	resp, err := adaptor.Client.ReceiveMessage(params)

	if err != nil {
		fmt.Println(err.Error())
		return
	}

	if len(resp.Messages) == 0 {
		return
	}

	item := resp.Messages[0]

	deleteParams := &sqs.DeleteMessageInput{
		QueueUrl:      aws.String(adaptor.QueueURL),
		ReceiptHandle: aws.String(*item.ReceiptHandle),
	}
	_, delerr := adaptor.Client.DeleteMessage(deleteParams)

	if delerr != nil {
		fmt.Println(err.Error())
		return
	}
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
