package exec

import (
	"encoding/json"
	"errors"
	"fmt"
	log "github.com/Sirupsen/logrus"
	"github.com/docker/engine-api/client"
	"github.com/docker/engine-api/types"
	"github.com/docker/engine-api/types/container"
	exemsg "github.com/operable/cogexec/messages"
	"github.com/operable/go-relay/relay/config"
	"github.com/operable/go-relay/relay/messages"
	"golang.org/x/net/context"
	"net/http"
	"time"
)

const (
	megabyte = 1024 * 1024
)

var timeout = time.Duration(60) * time.Second

// ErrorTimeout is returned when an execute request fails
// to respond within 1 minute
var ErrorTimeout = errors.New("Command execution timed out.")

// RelayCreatedDockerLabel is used to mark all containers
// started by Relay for later clean up.
var RelayCreatedDockerLabel = "io.operable.cog.relay.created"

// DockerEnvironment is an execution environment which runs inside a
// Docker container.
type DockerEnvironment struct {
	client      *client.Client
	container   string
	shortID     string
	relayConfig *config.Config
	bundle      *config.Bundle
}

// NewDockerEnvironment creates a new DockerEnvironment instance
func NewDockerEnvironment(relayConfig *config.Config, bundle *config.Bundle) (Environment, error) {
	client, err := newClient(relayConfig.Docker)
	if err != nil {
		return nil, err
	}
	de := &DockerEnvironment{
		client:      client,
		relayConfig: relayConfig,
		bundle:      bundle,
	}
	hostConfig, config := de.createContainerOptions()
	container, err := client.ContainerCreate(context.Background(), config, hostConfig, nil, "")
	if err != nil {
		return nil, err
	}
	de.container = container.ID
	de.shortID = shortContainerID(container.ID)
	log.Debugf("Created container %s with %d MB RAM.", de.shortID, (hostConfig.Memory / megabyte))
	err = de.client.ContainerStart(context.Background(), de.container, "")
	if err != nil {
		de.Terminate(true)
		return nil, err
	}
	log.Debugf("Started container %s.", de.shortID)
	return de, nil
}

// BundleName is required by the environment.Environment interface
func (de *DockerEnvironment) BundleName() string {
	return de.bundle.Name
}

// Terminate is required by the environment.Environment interface
func (de *DockerEnvironment) Terminate(kill bool) {
	if kill == true {
		de.client.ContainerRemove(context.Background(), de.container, types.ContainerRemoveOptions{
			RemoveVolumes: true,
			Force:         true,
		})
		return
	}
	start := time.Now()
	de.client.ContainerStop(context.Background(), de.container, 1)
	finish := time.Now()
	log.Debugf("Container %s terminated in %f seconds.", de.shortID,
		finish.Sub(start).Seconds())
}

// Execute is required by the environment.Environment interface
func (de *DockerEnvironment) Execute(request *messages.ExecutionRequest) ([]byte, []byte, error) {
	start := time.Now()
	conn, err := de.connectStreams()
	defer conn.Close()
	if err != nil {
		return EmptyResult, EmptyResult, err
	}
	req := de.prepareRequest(request)
	err = conn.Send(&req)
	if err != nil {
		return EmptyResult, EmptyResult, err
	}
	resp, err := conn.Receive()
	finish := time.Now()
	log.Infof("Docker container %s ran %s for %f secs.", de.shortID, request.Command, finish.Sub(start).Seconds())
	if err != nil {
		return EmptyResult, EmptyResult, err
	}
	return resp.Stdout, resp.Stderr, nil
}

func (de *DockerEnvironment) connectStreams() (*ContainerConnection, error) {
	conn, err := de.client.ContainerAttach(context.Background(), de.container, types.ContainerAttachOptions{
		Stream: true,
		Stdin:  true,
		Stdout: true,
	})
	if err != nil {
		return nil, err
	}
	return NewContainerConnection(conn), nil
}

func (de *DockerEnvironment) createContainerOptions() (*container.HostConfig, *container.Config) {
	imageID := fmt.Sprintf("%s:%s", de.bundle.Docker.Image, de.bundle.Docker.Tag)
	hc := &container.HostConfig{
		Privileged:  false,
		VolumesFrom: []string{"cogexec"},
	}
	hc.Memory = int64(de.relayConfig.Docker.ContainerMemory * megabyte)
	config := &container.Config{
		Image:     imageID,
		OpenStdin: true,
		StdinOnce: false,
		Labels: map[string]string{
			RelayCreatedDockerLabel: "yes",
		},
		Cmd: []string{"/operable/cogexec/bin/cogexec"},
	}
	return hc, config
}

func (de *DockerEnvironment) prepareRequest(request *messages.ExecutionRequest) exemsg.ExecCommandRequest {
	input, _ := json.Marshal(request.CogEnv)
	return exemsg.ExecCommandRequest{
		Executable: de.bundle.Commands[request.CommandName()].Executable,
		CogEnv:     input,
		Env:        BuildCallingEnvironment(request, de.relayConfig),
		Die:        false,
	}
}

func newClient(dockerConfig *config.DockerInfo) (*client.Client, error) {
	if dockerConfig.UseEnv {
		client, err := client.NewEnvClient()
		if err != nil {
			return nil, err
		}
		return client, nil
	}
	client, err := client.NewClient(dockerConfig.SocketPath, "", &http.Client{}, nil)
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
