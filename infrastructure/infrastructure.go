package infrastructure

import (
	"context"
	"fmt"
	"net"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/iam"
	"github.com/aws/aws-sdk-go/service/lambda"
	"github.com/aws/aws-sdk-go/service/sqs"
	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/network"
	"github.com/docker/docker/client"
	"github.com/goadapp/goad/version"
	"github.com/satori/go.uuid"
	try "gopkg.in/matryer/try.v1"
)

const (
	RabbitName    = "rabbitmq"
	RabbitPort    = "5672"
	RabbitRetries = 30
)

var RabbitURL = fmt.Sprintf("amqp://guest:guest@%s:%s/", RabbitName, RabbitPort)

// AwsInfrastructure manages the resource creation and updates necessary to use
// Goad.
type AwsInfrastructure struct {
	config   *aws.Config
	queueURL string
	regions  []string
}

type dockerInfrastructure struct {
	Cli                 *client.Client
	NetworkID           string
	RabbitMQContainerID string
	RabbitMQContainerIP string
}

type Infrastructure interface {
	Setup() (teardown func(), err error)
	GetQueueURL() string
}

// New creates the required infrastructure to run the load tests in Lambda
// functions.
func New(regions []string, config *aws.Config) Infrastructure {
	infra := &AwsInfrastructure{config: config, regions: regions}
	return infra
}

func handleErr(err error) {
	if err != nil {
		panic(err)
	}
}

func NewDockerInfrastructure() Infrastructure {
	infra := &dockerInfrastructure{}
	cli, err := client.NewEnvClient()
	handleErr(err)
	infra.Cli = cli
	return infra
}

func (i *dockerInfrastructure) Setup() (func(), error) {
	ctx := context.Background()
	cli := i.Cli
	list, err := cli.NetworkList(ctx, types.NetworkListOptions{})
	for _, network := range list {
		if network.Name == "goad-bridge" {
			i.NetworkID = network.ID
		}
	}
	handleErr(err)

	if i.NetworkID == "" {
		netw, nerr := cli.NetworkCreate(ctx, "goad-bridge", types.NetworkCreate{
			CheckDuplicate: true,
			// IPAM: &network.IPAM{
			// 	Config: []network.IPAMConfig{network.IPAMConfig{
			//
			// 	// Subnet: "10.0.0.0/16",
			// 	}},
			// },
		})
		handleErr(nerr)
		i.NetworkID = netw.ID
	}

	running := false
	containers, err := cli.ContainerList(ctx, types.ContainerListOptions{
		All: true,
	})
	handleErr(err)
	for _, container := range containers {
		if strings.Contains(container.Image, "rabbitmq:") {
			i.RabbitMQContainerID = container.ID
			if container.State == "running" {
				i.RabbitMQContainerIP = i.getRabbitContainerIP()
				running = true
			}
		}
	}

	if i.RabbitMQContainerID == "" {
		// Create container to execute lambda
		resp, cerr := cli.ContainerCreate(ctx, &container.Config{
			Image: "rabbitmq:3",
		}, &container.HostConfig{
			AutoRemove: true,
		}, &network.NetworkingConfig{
			EndpointsConfig: map[string]*network.EndpointSettings{
				"goad-bridge": &network.EndpointSettings{
				// IPAddress: i.RabbitMQContainerIP,
				},
			},
		}, "rabbitmq")
		handleErr(cerr)
		i.RabbitMQContainerID = resp.ID
	}

	if !running {
		// run container
		err = cli.ContainerStart(ctx, i.RabbitMQContainerID, types.ContainerStartOptions{})
		handleErr(err)

		ip := i.getRabbitContainerIP()
		i.RabbitMQContainerIP = ip
		err = try.Do(func(attemt int) (bool, error) {
			conn, derr := net.Dial("tcp", fmt.Sprintf("%s:%s", ip, RabbitPort))
			if derr != nil {
				time.Sleep(1 * time.Second)
				return attemt < RabbitRetries, derr
			}
			defer conn.Close()
			return true, nil
		})
		handleErr(err)
	}

	return i.Teardown, nil
}

func (i *dockerInfrastructure) getRabbitContainerIP() string {
	ctx := context.Background()
	cli := i.Cli
	ip := ""
	err := try.Do(func(attemt int) (bool, error) {
		data, derr := cli.ContainerInspect(ctx, i.RabbitMQContainerID)
		if derr != nil {
			time.Sleep(1 * time.Second)
			return attemt < RabbitRetries, derr
		}
		networks := data.NetworkSettings.Networks
		ip = networks["goad-bridge"].IPAddress
		return true, nil
	})
	handleErr(err)
	return ip
}

func (i *dockerInfrastructure) GetQueueURL() string {
	return fmt.Sprintf("amqp://guest:guest@%s:%s/", i.RabbitMQContainerIP, RabbitPort)
}

func (i *dockerInfrastructure) Teardown() {
	ctx := context.Background()
	cli, err := client.NewEnvClient()
	handleErr(err)

	timeout := time.Second * 1

	// log.Printf("stopping RabbitMQ: %s", i.RabbitMQContainerID)
	err = cli.ContainerStop(ctx, i.RabbitMQContainerID, &timeout)
	handleErr(err)

	containers, err := cli.ContainerList(ctx, types.ContainerListOptions{
		All: true,
	})
	handleErr(err)
	for _, container := range containers {
		if strings.Contains(container.Image, "lambci/lambda") {
			cli.ContainerStop(ctx, container.ID, &timeout)
		}
	}

	// log.Printf("shutting down bridge-network: %s", i.NetworkID)
	err = cli.NetworkRemove(ctx, i.NetworkID)
	handleErr(err)
}

// QueueURL returns the URL of the SQS queue to use for the load test session.
func (infra *AwsInfrastructure) GetQueueURL() string {
	return infra.queueURL
}

// Clean removes any AWS resources that cannot be reused for a subsequent test.
func (infra *AwsInfrastructure) Teardown() {
	infra.removeSQSQueue()
}

func (infra *AwsInfrastructure) Setup() (func(), error) {
	roleArn, err := infra.createIAMLambdaRole("goad-lambda-role")
	if err != nil {
		return nil, err
	}
	zip, err := Asset("data/lambda.zip")
	if err != nil {
		return nil, err
	}

	for _, region := range infra.regions {
		err = infra.createOrUpdateLambdaFunction(region, roleArn, zip)
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

func (infra *AwsInfrastructure) createOrUpdateLambdaFunction(region, roleArn string, payload []byte) error {
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

func (infra *AwsInfrastructure) createLambdaFunction(svc *lambda.Lambda, roleArn string, payload []byte) error {
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

func (infra *AwsInfrastructure) updateLambdaFunction(svc *lambda.Lambda, roleArn string, payload []byte) error {
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

func (infra *AwsInfrastructure) createIAMLambdaRole(roleName string) (arn string, err error) {
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

func (infra *AwsInfrastructure) createIAMLambdaRolePolicy(roleName string) error {
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
				 "Effect": "Allow",
				 "Action": [
						 "lambda:Invoke*"
				 ],
				 "Resource": [
						 "arn:aws:lambda:*:*:goad:*"
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
	svc := sqs.New(session.New(), infra.config)

	resp, err := svc.CreateQueue(&sqs.CreateQueueInput{
		QueueName: aws.String("goad-" + uuid.NewV4().String()),
	})

	if err != nil {
		return "", err
	}

	return *resp.QueueUrl, nil
}

func (infra *AwsInfrastructure) removeSQSQueue() {
	svc := sqs.New(session.New(), infra.config)

	svc.DeleteQueue(&sqs.DeleteQueueInput{
		QueueUrl: aws.String(infra.queueURL),
	})
}
