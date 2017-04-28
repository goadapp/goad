package queue

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
	Instance         string `json:"instance"`
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

// NewSQSAdapter returns a new sqs adator object
func NewSQSAdapter(awsConfig *aws.Config, queueURL string) *SQSAdaptor {
	return &SQSAdaptor{getClient(awsConfig), queueURL}
}

// NewDummyAdaptor returns a new sqs adator object
func NewDummyAdaptor(queueURL string) *DummyAdaptor {
	return &DummyAdaptor{queueURL}
}

func getClient(awsConfig *aws.Config) *sqs.SQS {
	client := sqs.New(session.New(), awsConfig)
	return client
}

// Receive a result, or timeout in 1 second
func (adaptor SQSAdaptor) Receive() *AggData {
	params := &sqs.ReceiveMessageInput{
		QueueUrl:            aws.String(adaptor.QueueURL),
		MaxNumberOfMessages: aws.Int64(1),
		VisibilityTimeout:   aws.Int64(1),
		WaitTimeSeconds:     aws.Int64(1),
	}
	resp, err := adaptor.Client.ReceiveMessage(params)

	if err != nil {
		fmt.Println(err.Error())
		return nil
	}

	if len(resp.Messages) == 0 {
		return nil
	}

	item := resp.Messages[0]

	deleteParams := &sqs.DeleteMessageInput{
		QueueUrl:      aws.String(adaptor.QueueURL),
		ReceiptHandle: aws.String(*item.ReceiptHandle),
	}
	_, delerr := adaptor.Client.DeleteMessage(deleteParams)

	if delerr != nil {
		fmt.Println(err.Error())
		return nil
	}

	result, jsonerr := resultFromJSON(*item.Body)
	if jsonerr != nil {
		fmt.Println(err.Error())
		return nil
	}

	return &result
}

func resultFromJSON(str string) (AggData, error) {
	var result AggData
	jsonerr := json.Unmarshal([]byte(str), &result)
	if jsonerr != nil {
		return result, jsonerr
	}
	return result, nil
}

func jsonFromResult(result AggData) (string, error) {
	data, jsonerr := json.Marshal(result)
	if jsonerr != nil {
		return "", jsonerr
	}
	return string(data), nil
}

// SendResult adds a result to the queue
func (adaptor SQSAdaptor) SendResult(result AggData) {
	str, jsonerr := jsonFromResult(result)
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
func (adaptor DummyAdaptor) SendResult(result AggData) {
	str, jsonerr := jsonFromResult(result)
	if jsonerr != nil {
		fmt.Println(jsonerr)
		return
	}
	fmt.Println("\n" + str)
}
