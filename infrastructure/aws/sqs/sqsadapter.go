package sqs

import (
	"encoding/json"
	"fmt"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/sqs"
	"github.com/goadapp/goad/api"
)

// Adapter is used to send messages to the queue
type Adapter struct {
	Client   *sqs.SQS
	QueueURL string
}

// DummyAdapter is used to send messages to the screen for testing
type DummyAdapter struct {
	QueueURL string
}

// NewSQSAdapter returns a new sqs adator object
func NewSQSAdapter(awsConfig *aws.Config, queueURL string) *Adapter {
	return &Adapter{getClient(awsConfig), queueURL}
}

// NewDummyAdaptor returns a new sqs adator object
func NewDummyAdaptor(queueURL string) *DummyAdapter {
	return &DummyAdapter{queueURL}
}

func getClient(awsConfig *aws.Config) *sqs.SQS {
	client := sqs.New(session.New(), awsConfig)
	return client
}

// Receive a result, or timeout in 1 second
func (adaptor Adapter) Receive() *api.RunnerResult {
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
	return result
}

func resultFromJSON(str string) (*api.RunnerResult, error) {
	var result = &api.RunnerResult{
		Statuses: make(map[string]int),
	}
	jsonerr := json.Unmarshal([]byte(str), result)
	if jsonerr != nil {
		return result, jsonerr
	}
	return result, nil
}

func jsonFromResult(result api.RunnerResult) (string, error) {
	data, jsonerr := json.Marshal(result)
	if jsonerr != nil {
		return "", jsonerr
	}
	return string(data), nil
}

// SendResult adds a result to the queue
func (adaptor Adapter) SendResult(result api.RunnerResult) error {
	str, jsonerr := jsonFromResult(result)
	if jsonerr != nil {
		fmt.Println(jsonerr)
		panic(jsonerr)
	}
	params := &sqs.SendMessageInput{
		MessageBody: aws.String(str),
		QueueUrl:    aws.String(adaptor.QueueURL),
	}
	_, err := adaptor.Client.SendMessage(params)

	return err
}

// SendResult prints the result
func (adaptor DummyAdapter) SendResult(result api.RunnerResult) {
	str, jsonerr := jsonFromResult(result)
	if jsonerr != nil {
		fmt.Println(jsonerr)
		return
	}
	fmt.Println("\n" + str)
}
