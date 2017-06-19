package dockerinfra

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"os"
	"strings"
	"time"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/network"
	"github.com/docker/docker/client"
	"github.com/goadapp/goad/infrastructure"
	"github.com/spf13/afero"
	try "gopkg.in/matryer/try.v1"
)

const (
	rabbitPort    = "5672"
	rabbitRetries = 30
)

type dockerInfrastructure struct {
	Cli                 *client.Client
	settings            infrastructure.Settings
	NetworkID           string
	RabbitMQContainerID string
	RabbitMQContainerIP string
}

func handleErr(err error) {
	if err != nil {
		panic(err)
	}
}

func NewDockerInfrastructure() infrastructure.Infrastructure {
	cli, err := client.NewEnvClient()
	handleErr(err)
	infra := &dockerInfrastructure{
		Cli: cli,
	}
	DockerPullLambdaImage()
	DockerPullRabbitMQImage()
	return infra
}

func (i *dockerInfrastructure) Run(args infrastructure.InvokeArgs) {
	i.runAsDockerContainer(args)
}

func (i *dockerInfrastructure) Setup(settings infrastructure.Settings) (func(), error) {
	i.settings = settings
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
				"goad-bridge": &network.EndpointSettings{},
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
		fmt.Print("Waiting for queue to get ready")
		try.MaxRetries = rabbitRetries
		err = try.Do(func(attempt int) (bool, error) {
			conn, derr := net.Dial("tcp", fmt.Sprintf("%s:%s", ip, rabbitPort))
			if derr != nil {
				fmt.Print(".")
				time.Sleep(1 * time.Second)
				return true, derr
			}
			defer conn.Close()
			return true, nil
		})
		handleErr(err)
	}

	return i.Teardown, nil
}

func DockerPullLambdaImage() {
	DockerPullImage("lambci/lambda")
}

func DockerPullRabbitMQImage() {
	DockerPullImage("rabbitmq:3")
}

func DockerPullImage(imageName string) {
	ctx := context.Background()
	cli, err := client.NewEnvClient()
	handleErr(err)

	// Pull the image from dockerhub.
	out, err := cli.ImagePull(ctx, imageName, types.ImagePullOptions{})
	handleErr(err)
	defer out.Close()
	io.Copy(os.Stdout, out)
}

func (i *dockerInfrastructure) getRabbitContainerIP() string {
	ctx := context.Background()
	cli := i.Cli
	ip := ""
	try.MaxRetries = rabbitRetries
	err := try.Do(func(attempt int) (bool, error) {
		data, derr := cli.ContainerInspect(ctx, i.RabbitMQContainerID)
		if derr != nil {
			time.Sleep(1 * time.Second)
			return true, derr
		}
		networks := data.NetworkSettings.Networks
		ip = networks["goad-bridge"].IPAddress
		return true, nil
	})
	handleErr(err)
	return ip
}

func (i *dockerInfrastructure) runAsDockerContainer(args interface{}) {
	ctx := context.Background()
	cli, err := client.NewEnvClient()
	handleErr(err)
	rabbitmqURL := fmt.Sprintf("RABBITMQ=%s", i.GetQueueURL())

	var runnerPath string
	if i.settings.RunnerPath == "" {
		runnerPath = createTempDefaultRunner()
	} else {
		runnerPath = os.ExpandEnv(fmt.Sprintf("${PWD}/%s", i.settings.RunnerPath))
	}
	// Create container to execute lambda
	resp, err := cli.ContainerCreate(ctx, &container.Config{
		Image: "lambci/lambda",
		Cmd:   append([]string{"index.handler"}, toJSONString(args)),
		Volumes: map[string]struct{}{
			"/var/task": struct{}{},
		},
		Env: []string{rabbitmqURL},
	}, &container.HostConfig{
		AutoRemove: true,
		Binds: []string{
			fmt.Sprintf("%s:/var/task:ro", runnerPath),
		},
	}, &network.NetworkingConfig{
		EndpointsConfig: map[string]*network.EndpointSettings{
			"goad-bridge": &network.EndpointSettings{},
		},
	}, "")
	handleErr(err)

	// run container
	err = cli.ContainerStart(ctx, resp.ID, types.ContainerStartOptions{})
	handleErr(err)
}

func toJSONString(args interface{}) string {
	b, err := json.Marshal(args)
	handleErr(err)
	return string(b[:])
}

func (i *dockerInfrastructure) GetQueueURL() string {
	return fmt.Sprintf("amqp://guest:guest@%s:%s/", i.RabbitMQContainerIP, rabbitPort)
}

func (i *dockerInfrastructure) Teardown() {
	ctx := context.Background()
	cli, err := client.NewEnvClient()
	handleErr(err)

	timeout := time.Second * 1

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

	err = cli.NetworkRemove(ctx, i.NetworkID)
	handleErr(err)
}

func createTempDefaultRunner() string {
	var fs afero.Fs = afero.NewOsFs()
	dir := afero.GetTempDir(fs, "defaultRunner")
	defaultRunnerZip, _ := infrastructure.Asset(infrastructure.DefaultRunnerAsset)
	infrastructure.Unzip(defaultRunnerZip, dir)

	return dir
}
