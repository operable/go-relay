package exec

import (
	"bytes"
	"encoding/json"
	"fmt"
	log "github.com/Sirupsen/logrus"
	"github.com/fsouza/go-dockerclient"
	"github.com/operable/go-relay/relay/config"
	"github.com/operable/go-relay/relay/messages"
	"io"
	"time"
)

// RelayCreatedDockerLabel is used to mark all containers
// started by Relay for later clean up.
var RelayCreatedDockerLabel = "io.operable.cog.relay.created"

// DockerEnvironment is an execution environment which runs inside a
// Docker container.
type DockerEnvironment struct {
	client      *docker.Client
	relayConfig *config.Config
	bundle      *config.Bundle
}

// NewDockerEnvironment creates a new DockerEnvironment instance
func NewDockerEnvironment(relayConfig *config.Config, bundle *config.Bundle) (Environment, error) {
	client, err := newClient(relayConfig.Docker)
	if err != nil {
		return nil, err
	}
	return &DockerEnvironment{
		client:      client,
		relayConfig: relayConfig,
		bundle:      bundle,
	}, nil
}

func (de *DockerEnvironment) Execute(request *messages.ExecutionRequest) ([]byte, []byte, error) {
	createOptions, err := de.createContainerOptions(request)
	if err != nil {
		return EmptyResult, EmptyResult, err
	}
	container, err := de.client.CreateContainer(*createOptions)
	if err != nil {
		return EmptyResult, EmptyResult, err
	}
	containerID := shortContainerID(container.ID)
	input, _ := json.Marshal(request.CogEnv)
	stdout := bytes.NewBuffer([]byte{})
	stderr := bytes.NewBuffer([]byte{})
	inputWaiter, err := de.attachInputWriter(container.ID, input)
	if err != nil {
		de.client.KillContainer(docker.KillContainerOptions{
			ID: container.ID,
		})
		return EmptyResult, EmptyResult, err
	}
	outputWaiter, err := de.attachOutputReader(container.ID, stdout, stderr)
	if err != nil {
		inputWaiter.Close()
		de.client.KillContainer(docker.KillContainerOptions{
			ID: container.ID,
		})
		return EmptyResult, EmptyResult, err
	}
	start := time.Now()
	err = de.client.StartContainer(container.ID, nil)
	if err != nil {
		inputWaiter.Close()
		outputWaiter.Close()
		return EmptyResult, EmptyResult, err
	}
	de.client.WaitContainer(container.ID)
	finish := time.Now()
	log.Infof("Docker container %s ran %s for %f secs.", containerID, request.Command, finish.Sub(start).Seconds())
	return stdout.Bytes(), stderr.Bytes(), nil
}

func (de *DockerEnvironment) createContainerOptions(request *messages.ExecutionRequest) (*docker.CreateContainerOptions, error) {
	imageID := fmt.Sprintf("%s:%s", de.bundle.Docker.Image, de.bundle.Docker.Tag)
	command := request.CommandName()
	return &docker.CreateContainerOptions{
		Name: "",
		Config: &docker.Config{
			Image:      imageID,
			Env:        BuildCallingEnvironment(request, de.relayConfig),
			Memory:     64 * 1024 * 1024, // 64MB
			MemorySwap: 0,
			StdinOnce:  true,
			OpenStdin:  true,
			Labels: map[string]string{
				RelayCreatedDockerLabel: "yes",
			},
			Cmd: []string{de.bundle.Commands[command].Executable},
		},
		HostConfig: &docker.HostConfig{
			Privileged: false,
		},
	}, nil
}

func (de *DockerEnvironment) attachInputWriter(containerID string, input []byte) (docker.CloseWaiter, error) {
	client, _ := newClient(de.relayConfig.Docker)
	return client.AttachToContainerNonBlocking(docker.AttachToContainerOptions{
		Container:   containerID,
		InputStream: bytes.NewBuffer(input),
		Stdin:       true,
		Stream:      true,
	})
}

func (de *DockerEnvironment) attachOutputReader(containerID string, stdout io.Writer, stderr io.Writer) (docker.CloseWaiter, error) {
	client, _ := newClient(de.relayConfig.Docker)
	return client.AttachToContainerNonBlocking(docker.AttachToContainerOptions{
		Container:    containerID,
		Stdout:       true,
		Stderr:       true,
		Stream:       true,
		OutputStream: stdout,
		ErrorStream:  stderr,
	})
}

func newClient(dockerConfig *config.DockerInfo) (*docker.Client, error) {
	if dockerConfig.UseEnv {
		client, err := docker.NewClientFromEnv()
		if err != nil {
			return nil, err
		}
		return client, nil
	}
	client, err := docker.NewClient(dockerConfig.SocketPath)
	if err != nil {
		return nil, err
	}
	return client, nil
}

func shortContainerID(containerID string) string {
	idEnd := len(containerID)
	idStart := idEnd - 12
	return containerID[idStart:idEnd]
}
