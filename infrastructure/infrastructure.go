package infrastructure

import (
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/iam"
	"github.com/aws/aws-sdk-go/service/lambda"
	"github.com/aws/aws-sdk-go/service/sqs"
	"github.com/goadapp/goad/version"
	"github.com/satori/go.uuid"
)

// Infrastructure manages the resource creation and updates necessary to use
// Goad.
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
		err = infra.createOrUpdateLambdaFunction(region, roleArn, zip)
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

func (infra *Infrastructure) createOrUpdateLambdaFunction(region, roleArn string, payload []byte) error {
	config := aws.NewConfig().WithRegion(region)
	svc := lambda.New(session.New(), config)

	exists, err := lambdaExists(svc)

	if err != nil {
		return err
	}

	if exists {
		aliasExists, err := lambdaAliasExists(svc)
		if err != nil || aliasExists {
			return err
		}
		return infra.updateLambdaFunction(svc, roleArn, payload)
	}

	return infra.createLambdaFunction(svc, roleArn, payload)
}

func (infra *Infrastructure) createLambdaFunction(svc *lambda.Lambda, roleArn string, payload []byte) error {
	function, err := svc.CreateFunction(&lambda.CreateFunctionInput{
		Code: &lambda.FunctionCode{
			ZipFile: payload,
		},
		FunctionName: aws.String("goad"),
		Handler:      aws.String("index.handler"),
		Role:         aws.String(roleArn),
		Runtime:      aws.String("nodejs4.3"),
		MemorySize:   aws.Int64(1536),
		Publish:      aws.Bool(true),
		Timeout:      aws.Int64(300),
	})
	if err != nil {
		return err
	}
	return createLambdaAlias(svc, function.Version)
}

func (infra *Infrastructure) updateLambdaFunction(svc *lambda.Lambda, roleArn string, payload []byte) error {
	function, err := svc.UpdateFunctionCode(&lambda.UpdateFunctionCodeInput{
		ZipFile:      payload,
		FunctionName: aws.String("goad"),
		Publish:      aws.Bool(true),
	})
	if err != nil {
		return err
	}
	return createLambdaAlias(svc, function.Version)
}

func lambdaExists(svc *lambda.Lambda) (bool, error) {
	_, err := svc.GetFunction(&lambda.GetFunctionInput{
		FunctionName: aws.String("goad"),
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

func createLambdaAlias(svc *lambda.Lambda, functionVersion *string) error {
	_, err := svc.CreateAlias(&lambda.CreateAliasInput{
		FunctionName:    aws.String("goad"),
		FunctionVersion: functionVersion,
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

func (infra *Infrastructure) createIAMLambdaRole(roleName string) (arn string, err error) {
	svc := iam.New(session.New(), infra.config)

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
