package awsinfra

import (
	"bytes"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"os"
	"path"
	"time"

	"github.com/goadapp/goad/goad/types"
	"github.com/goadapp/goad/infrastructure/aws/sqsadapter"

	"github.com/Songmu/prompter"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/iam"
	"github.com/aws/aws-sdk-go/service/lambda"
	"github.com/aws/aws-sdk-go/service/sqs"
	"github.com/goadapp/goad/infrastructure"
	"github.com/goadapp/goad/result"
	"github.com/goadapp/goad/version"
	uuid "github.com/satori/go.uuid"
)

var (
	roleDate = time.Date(2017, 07, 13, 18, 40, 0, 0, time.UTC)
)

// AwsInfrastructure manages the resource creation and updates necessary to use
// Goad.
type AwsInfrastructure struct {
	config    *types.TestConfig
	awsConfig *aws.Config
	queueURL  string
}

// New creates the required infrastructure to run the load tests in Lambda
// functions.
func New(config *types.TestConfig) infrastructure.Infrastructure {
	awsConfig := aws.NewConfig().WithRegion(config.Regions[0])
	infra := &AwsInfrastructure{config: config, awsConfig: awsConfig}
	return infra
}

func (infra *AwsInfrastructure) GetSettings() *types.TestConfig {
	return infra.config
}

// GetQueueURL returns the URL of the SQS queue to use for the load test session
func (infra *AwsInfrastructure) GetQueueURL() string {
	return infra.queueURL
}

func (infra *AwsInfrastructure) Receive(results chan *result.LambdaResults) {
	defer close(results)
	data := result.SetupRegionsAggData(infra.config.Lambdas)
	adaptor := sqsadapter.New(infra.awsConfig, infra.GetQueueURL())

	timeoutStart := time.Now()
	for {
		lambdaResults := adaptor.Receive()
		if lambdaResults != nil {
			for _, lambdaResult := range lambdaResults {
				lambdaAggregate := &data.Lambdas[lambdaResult.RunnerID]
				result.AddResult(lambdaAggregate, lambdaResult)
				results <- data
			}
			if data.AllLambdasFinished() {
				break
			}
			timeoutStart = time.Now()
		} else {
			waited := time.Since(timeoutStart)
			if waited.Seconds() > 20 {
				break
			}
		}
	}
}

func (infra *AwsInfrastructure) Run(args infrastructure.InvokeArgs) {
	infra.invokeLambda(args)
}

func (infra *AwsInfrastructure) invokeLambda(args interface{}) {
	svc := lambda.New(session.New(), infra.awsConfig)

	svc.Invoke(&lambda.InvokeInput{
		FunctionName: aws.String("goad"),
		Payload:      toByteArray(args),
	})
}

func toByteArray(args interface{}) []byte {
	j, err := json.Marshal(args)
	handleErr(err)
	return j
}

func handleErr(err error) {
	if err != nil {
		panic(err)
	}
}

// teardown removes any AWS resources that cannot be reused for a subsequent
// test
func (infra *AwsInfrastructure) teardown() {
	infra.removeSQSQueue()
}

func (infra *AwsInfrastructure) Setup() (func(), error) {
	roleArn, err := infra.createIAMLambdaRole("goad-lambda-role")
	if err != nil {
		return nil, err
	}

	var zipBuffer bytes.Buffer
	if infra.config.RunnerPath != "" {
		runnerPath := fmt.Sprintf("%s/", path.Clean(infra.config.RunnerPath))
		err = infrastructure.Zipit(runnerPath, &zipBuffer)
		if err != nil {
			return nil, err
		}
	} else {
		assetBytes, assetErr := infrastructure.Asset(infrastructure.DefaultRunnerAsset)
		if assetErr != nil {
			return nil, assetErr
		}
		zipBuffer = *bytes.NewBuffer(assetBytes)
	}

	for _, region := range infra.config.Regions {
		err = infra.createOrUpdateLambdaFunction(region, roleArn, zipBuffer.Bytes())
		if err != nil {
			return nil, err
		}
	}
	queueURL, err := infra.createSQSQueue()
	if err != nil {
		return nil, err
	}
	infra.queueURL = queueURL
	return func() {

	}, nil
}

func calcShasum(payload []byte) string {
	h := sha256.Sum256(payload)
	return string(base64.StdEncoding.EncodeToString(h[:]))
}

func (infra *AwsInfrastructure) createOrUpdateLambdaFunction(region, roleArn string, payload []byte) error {
	config := aws.NewConfig().WithRegion(region)
	svc := lambda.New(session.New(), config)

	exists, err := lambdaExists(svc)
	if err != nil {
		return err
	}
	if !exists {
		return infra.createLambdaFunction(svc, roleArn, payload)
	}
	upToDate, err := lambdaUpToDate(svc, calcShasum(payload))
	if err != nil {
		return err
	}
	if !upToDate {
		return infra.updateLambdaFunction(svc, roleArn, payload)
	}
	return nil
}

func (infra *AwsInfrastructure) createLambdaFunction(svc *lambda.Lambda, roleArn string, payload []byte) error {
	function, err := svc.CreateFunction(&lambda.CreateFunctionInput{
		Code: &lambda.FunctionCode{
			ZipFile: payload,
		},
		FunctionName: aws.String("goad"),
		Handler:      aws.String("index.handler"),
		Role:         aws.String(roleArn),
		Runtime:      aws.String("nodejs8.10"),
		MemorySize:   aws.Int64(1536),
		Publish:      aws.Bool(true),
		Timeout:      aws.Int64(300),
	})
	if err != nil {
		return err
	}
	return createOrUpdateLambdaAlias(svc, function.Version)
}

func (infra *AwsInfrastructure) updateLambdaFunction(svc *lambda.Lambda, roleArn string, payload []byte) error {
	function, err := svc.UpdateFunctionCode(&lambda.UpdateFunctionCodeInput{
		ZipFile:      payload,
		FunctionName: aws.String("goad"),
		Publish:      aws.Bool(true),
	})
	if err != nil {
		return err
	}
	return createOrUpdateLambdaAlias(svc, function.Version)
}

func lambdaExists(svc *lambda.Lambda) (bool, error) {
	_, err := svc.GetFunctionConfiguration(&lambda.GetFunctionConfigurationInput{
		FunctionName: aws.String("goad"),
	})
	notFound, err := checkResourseNotFound(err)
	if err != nil {
		return false, err
	}
	if notFound {
		return false, nil
	}
	return true, nil
}

func lambdaUpToDate(svc *lambda.Lambda, shasum string) (bool, error) {
	config, err := svc.GetFunctionConfiguration(&lambda.GetFunctionConfigurationInput{
		FunctionName: aws.String("goad"),
	})
	if err != nil {
		return false, err
	}
	return *config.CodeSha256 == shasum, nil
}

func checkResourseNotFound(err error) (bool, error) {
	if err != nil {
		if awsErr, ok := err.(awserr.Error); ok {
			if awsErr.Code() == "ResourceNotFoundException" {
				return true, nil
			}
		}
		return false, err
	}
	return false, err
}

func createOrUpdateLambdaAlias(svc *lambda.Lambda, functionVersion *string) error {
	_, err := svc.GetAlias(&lambda.GetAliasInput{
		FunctionName: aws.String("goad"),
		Name:         aws.String(version.LambdaVersion()),
	})
	if err != nil {
		return createLambdaAlias(svc, functionVersion)
	}
	return updateLambdaAlias(svc, functionVersion)
}

func createLambdaAlias(svc *lambda.Lambda, functionVersion *string) error {
	_, err := svc.CreateAlias(&lambda.CreateAliasInput{
		FunctionName:    aws.String("goad"),
		FunctionVersion: aws.String("$LATEST"),
		Name:            aws.String(version.LambdaVersion()),
	})
	return err
}

func updateLambdaAlias(svc *lambda.Lambda, functionVersion *string) error {
	_, err := svc.UpdateAlias(&lambda.UpdateAliasInput{
		FunctionName:    aws.String("goad"),
		FunctionVersion: aws.String("$LATEST"),
		Name:            aws.String(version.LambdaVersion()),
	})
	return err
}

func lambdaAliasExists(svc *lambda.Lambda) (bool, error) {
	_, err := svc.GetAlias(&lambda.GetAliasInput{
		FunctionName: aws.String("goad"),
		Name:         aws.String(version.LambdaVersion()),
	})

	if err != nil {
		if awsErr, ok := err.(awserr.Error); ok {
			if awsErr.Code() == "ResourceNotFoundException" {
				return false, nil
			}
		}
		return false, err
	}

	return true, nil
}

func (infra *AwsInfrastructure) createIAMLambdaRole(roleName string) (arn string, err error) {
	svc := iam.New(session.New(), infra.awsConfig)

	resp, err := svc.GetRole(&iam.GetRoleInput{
		RoleName: aws.String(roleName),
	})
	if err != nil {
		if awsErr, ok := err.(awserr.Error); ok {
			if awsErr.Code() == "NoSuchEntity" {
				res, err := svc.CreateRole(&iam.CreateRoleInput{
					AssumeRolePolicyDocument: aws.String(`{
        	          "Version": "2012-10-17",
        	          "Statement": {
        	            "Effect": "Allow",
        	            "Principal": {"Service": "lambda.amazonaws.com"},
        	            "Action": "sts:AssumeRole"
        	          }
            	    }`),
					RoleName: aws.String(roleName),
					Path:     aws.String("/"),
				})
				if err != nil {
					return "", err
				}
				if err := infra.createIAMLambdaRolePolicy(*res.Role.RoleName); err != nil {
					return "", err
				}
				return *res.Role.Arn, nil
			}
		}
		return "", err
	}

	CheckRoleDate(resp.Role)
	return *resp.Role.Arn, nil
}

func CheckRoleDate(role *iam.Role) {
	if role.CreateDate.Before(roleDate) {
		if !prompter.YN("Your IAM role for goad might be outdated, continue anyways?", true) {
			os.Exit(0)
		}
	}
}

func (infra *AwsInfrastructure) createIAMLambdaRolePolicy(roleName string) error {
	svc := iam.New(session.New(), infra.awsConfig)

	_, err := svc.PutRolePolicy(&iam.PutRolePolicyInput{
		PolicyDocument: aws.String(`{
          "Version": "2012-10-17",
          "Statement": [
					{
				 "Action": [
						 "sqs:SendMessage"
				 ],
				 "Effect": "Allow",
				 "Resource": "arn:aws:sqs:*:*:goad-*"
		 },
		 {
				 "Effect": "Allow",
				 "Action": [
						 "lambda:Invoke*"
				 ],
				 "Resource": [
						 "arn:aws:lambda:*:*:function:goad"
				 ]
		 },
			{
              "Action": [
                "logs:CreateLogGroup",
                "logs:CreateLogStream",
                "logs:PutLogEvents"
              ],
              "Effect": "Allow",
              "Resource": "arn:aws:logs:*:*:*"
	        }
          ]
        }`),
		PolicyName: aws.String("goad-lambda-role-policy"),
		RoleName:   aws.String(roleName),
	})
	return err
}

func (infra *AwsInfrastructure) createSQSQueue() (url string, err error) {
	svc := sqs.New(session.New(), infra.awsConfig)

	resp, err := svc.CreateQueue(&sqs.CreateQueueInput{
		QueueName: aws.String("goad-" + uuid.NewV4().String() + ".fifo"),
		Attributes: map[string]*string{
			"FifoQueue": aws.String("true"),
		},
	})

	if err != nil {
		return "", err
	}
	return *resp.QueueUrl, nil
}

func (infra *AwsInfrastructure) removeSQSQueue() {
	svc := sqs.New(session.New(), infra.awsConfig)

	svc.DeleteQueue(&sqs.DeleteQueueInput{
		QueueUrl: aws.String(infra.queueURL),
	})
}
