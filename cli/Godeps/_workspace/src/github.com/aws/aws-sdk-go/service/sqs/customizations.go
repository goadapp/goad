package sqs

import "github.com/gophergala2016/goad/cli/Godeps/_workspace/src/github.com/aws/aws-sdk-go/aws/request"

func init() {
	initRequest = func(r *request.Request) {
		setupChecksumValidation(r)
	}
}
