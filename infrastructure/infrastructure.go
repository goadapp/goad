package infrastructure

import (
	"time"

	"github.com/gophergala2016/goad/Godeps/_workspace/src/github.com/aws/aws-sdk-go/aws"
	"github.com/gophergala2016/goad/Godeps/_workspace/src/github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/gophergala2016/goad/Godeps/_workspace/src/github.com/aws/aws-sdk-go/aws/session"
	"github.com/gophergala2016/goad/Godeps/_workspace/src/github.com/aws/aws-sdk-go/service/iam"
	"github.com/gophergala2016/goad/Godeps/_workspace/src/github.com/aws/aws-sdk-go/service/lambda"
	"github.com/gophergala2016/goad/Godeps/_workspace/src/github.com/aws/aws-sdk-go/service/sqs"
	"github.com/gophergala2016/goad/Godeps/_workspace/src/github.com/satori/go.uuid"
)

type Infrastructure struct {
	config   *aws.Config
	queueURL string
	regions  []string
}

// New creates the required infrastructure to run the load tests in Lambda
// functions.
func New(regions []string, config *aws.Config) (*Infrastructure, error) {
	infra := &Infrastructure{config: config, regions: regions}
	if err := infra.setup(); err != nil {
		return nil, err
	}
	return infra, nil
}

// QueueURL returns the URL of the SQS queue to use for the load test session.
func (infra *Infrastructure) QueueURL() string {
	return infra.queueURL
}

// Clean removes any AWS resources that cannot be reused for a subsequent test.
func (infra *Infrastructure) Clean() {
	infra.removeSQSQueue()
}

func (infra *Infrastructure) setup() error {
	roleArn, err := infra.createIAMLambdaRole("goad-lambda-role")
	if err != nil {
		return err
	}
	zip, err := Asset("data/lambda.zip")
	if err != nil {
		return err
	}
	for _, region := range infra.regions {
		err = infra.createLambdaFunction(region, roleArn, zip)
		if err != nil {
			return err
		}
	}
	queueURL, err := infra.createSQSQueue()
	if err != nil {
		return err
	}
	infra.queueURL = queueURL
	return nil
}

func (infra *Infrastructure) createLambdaFunction(region, roleArn string, payload []byte) error {
	config := new(aws.Config)
	*config = *infra.config
	config.WithRegion(region)
	svc := lambda.New(session.New(), config)

	_, err := svc.GetFunction(&lambda.GetFunctionInput{
		FunctionName: aws.String("goad"),
	})

	if err != nil {
		if awsErr, ok := err.(awserr.Error); ok {
			if awsErr.Code() == "ResourceNotFoundException" {
				_, err := svc.CreateFunction(&lambda.CreateFunctionInput{
					Code: &lambda.FunctionCode{
						ZipFile: payload,
					},
					FunctionName: aws.String("goad"),
					Handler:      aws.String("index.handler"),
					Role:         aws.String(roleArn),
					Runtime:      aws.String("nodejs"),
					MemorySize:   aws.Int64(1536),
					Publish:      aws.Bool(true),
					Timeout:      aws.Int64(300),
				})
				if err != nil {
					if awsErr, ok := err.(awserr.Error); ok {
						// Calling this function too soon after creating the role might
						// fail, so we should retry after a little while.
						// TODO: limit the number of retries.
						if awsErr.Code() == "InvalidParameterValueException" {
							time.Sleep(time.Second)
							return infra.createLambdaFunction(region, roleArn, payload)
						}
					}
					return err
				}
			}
		}
	}

	return nil
}

func (infra *Infrastructure) createIAMLambdaRole(roleName string) (arn string, err error) {
	svc := iam.New(session.New(), infra.config)

	resp, err := svc.GetRole(&iam.GetRoleInput{
		RoleName: aws.String(roleName),
	})
	if err != nil {
		if awsErr, ok := err.(awserr.Error); ok {
			if awsErr.Code() == "NoSuchEntity" {
				resp, err := svc.CreateRole(&iam.CreateRoleInput{
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
				if err := infra.createIAMLambdaRolePolicy(*resp.Role.RoleName); err != nil {
					return "", err
				}
				return *resp.Role.Arn, nil
			}
		} else {
			return "", err
		}
	}

	return *resp.Role.Arn, nil
}

func (infra *Infrastructure) createIAMLambdaRolePolicy(roleName string) error {
	svc := iam.New(session.New(), infra.config)

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

func (infra *Infrastructure) createSQSQueue() (url string, err error) {
	svc := sqs.New(session.New(), infra.config)

	resp, err := svc.CreateQueue(&sqs.CreateQueueInput{
		QueueName: aws.String("goad-" + uuid.NewV4().String()),
	})

	if err != nil {
		return "", err
	}

	return *resp.QueueUrl, nil
}

func (infra *Infrastructure) removeSQSQueue() {
	svc := sqs.New(session.New(), infra.config)

	svc.DeleteQueue(&sqs.DeleteQueueInput{
		QueueUrl: aws.String(infra.queueURL),
	})
}
