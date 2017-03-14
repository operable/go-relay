package circuit

import (
	"fmt"
	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"
	"github.com/operable/circuit-driver/api"
	"github.com/operable/circuit-driver/io"
	"golang.org/x/net/context"
)

type dockerEnvironment struct {
	containerID   string
	options       CreateEnvironmentOptions
	dockerOptions DockerEnvironmentOptions
	image         string
	tag           string
	userData      EnvironmentUserData
	isDead        bool
	requests      chan api.ExecRequest
	results       chan api.ExecResult
	control       chan byte
}

func (de *dockerEnvironment) init(options CreateEnvironmentOptions) error {
	de.options = options
	de.dockerOptions = options.DockerOptions
	de.isDead = false
	client := de.dockerOptions.Conn
	hostConfig := container.HostConfig{
		Privileged:  false,
		VolumesFrom: []string{de.dockerOptions.DriverInstance},
		Binds:       de.dockerOptions.Binds,
	}
	hostConfig.Memory = de.dockerOptions.Memory * 1024 * 1024
	fullName := fmt.Sprintf("%s:%s", de.dockerOptions.Image, de.dockerOptions.Tag)
	config := container.Config{
		Image:     fullName,
		Cmd:       []string{de.dockerOptions.DriverPath},
		OpenStdin: true,
		StdinOnce: false,
		Tty:       false,
	}
	container, err := client.ContainerCreate(context.Background(), &config, &hostConfig, nil, "")
	if err != nil {
		return err
	}
	de.containerID = container.ID
	err = client.ContainerStart(context.Background(), de.containerID, types.ContainerStartOptions{})
	if err != nil {
		return err
	}
	de.requests = make(chan api.ExecRequest)
	de.results = make(chan api.ExecResult)
	de.control = make(chan byte)
	go func() {
		de.runWorker()
	}()
	return nil
}

func (de *dockerEnvironment) runWorker() {
	resp, err := de.dockerOptions.Conn.ContainerAttach(context.Background(), de.containerID, types.ContainerAttachOptions{
		Stream: true,
		Stdin:  true,
		Stdout: true,
		Stderr: true,
	})
	if err != nil {
		panic(err)
	}
	encoder := api.WrapEncoder(resp.Conn)
	decoder := api.WrapDecoder(io.NewDockerStdoutReader(resp.Conn))
	for {
		select {
		case <-de.control:
			break
		case request := <-de.requests:
			if err := encoder.EncodeRequest(&request); err != nil {
				panic(err)
			}
			var result api.ExecResult
			if err := decoder.DecodeResult(&result); err != nil && err != io.EOF {
				panic(err)
			}
			de.results <- result
		}
	}
}

func (de *dockerEnvironment) GetKind() EnvironmentKind {
	return DockerKind
}

func (de *dockerEnvironment) SetUserData(data EnvironmentUserData) error {
	if de.isDead {
		return ErrorDeadEnvironment
	}
	de.userData = data
	return nil
}

func (de *dockerEnvironment) GetUserData() (EnvironmentUserData, error) {
	if de.isDead {
		return nil, ErrorDeadEnvironment
	}
	return de.userData, nil
}

func (de *dockerEnvironment) GetMetadata() EnvironmentMetadata {
	return EnvironmentMetadata{
		"bundle":    de.options.Bundle,
		"image":     de.dockerOptions.Image,
		"tag":       de.dockerOptions.Tag,
		"container": de.containerID,
	}
}

func (de *dockerEnvironment) Run(request api.ExecRequest) (api.ExecResult, error) {
	if de.isDead {
		return EmptyExecResult, ErrorDeadEnvironment
	}
	de.requests <- request
	result := <-de.results
	return result, nil
}

func (de *dockerEnvironment) Shutdown() error {
	if de.isDead {
		return ErrorDeadEnvironment
	}
	de.control <- 1
	removeOptions := types.ContainerRemoveOptions{
		Force: true,
	}
	err := de.dockerOptions.Conn.ContainerRemove(context.Background(), de.containerID, removeOptions)
	de.isDead = true
	return err
}
