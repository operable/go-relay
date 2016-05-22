package exec

import (
	"bytes"
	"encoding/gob"
	"encoding/json"
	"fmt"
	log "github.com/Sirupsen/logrus"
	"github.com/fsouza/go-dockerclient"
	exemsg "github.com/operable/cogexec/messages"
	"github.com/operable/go-relay/relay/config"
	"github.com/operable/go-relay/relay/messages"
	"time"
)

const (
	Megabyte = 1024 * 1024
)

// RelayCreatedDockerLabel is used to mark all containers
// started by Relay for later clean up.
var RelayCreatedDockerLabel = "io.operable.cog.relay.created"

// DockerEnvironment is an execution environment which runs inside a
// Docker container.
type DockerEnvironment struct {
	client       *docker.Client
	container    *docker.Container
	shortID      string
	relayConfig  *config.Config
	bundle       *config.Bundle
	encoder      *gob.Encoder
	stdin        *bytes.Buffer
	inputWaiter  docker.CloseWaiter
	decoder      *gob.Decoder
	stdout       *bytes.Buffer
	outputWaiter docker.CloseWaiter
	barrier      chan struct{}
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
		stdin:       bytes.NewBuffer([]byte{}),
		stdout:      bytes.NewBuffer([]byte{}),
		barrier:     make(chan struct{}),
	}
	createOptions, err := de.createContainerOptions()
	if err != nil {
		return nil, err
	}
	container, err := client.CreateContainer(*createOptions)
	if err != nil {
		return nil, err
	}
	de.container = container
	de.shortID = shortContainerID(container.ID)
	log.Debugf("Created container %s with %d MB RAM.", de.shortID, (createOptions.Config.Memory / Megabyte))
	err = de.client.StartContainer(de.container.ID, nil)
	if err != nil {
		de.Terminate(true)
		return nil, err
	}
	if err = de.connectStreams(); err != nil {
		de.Terminate(true)
		return nil, err
	}
	return de, nil
}

func (de *DockerEnvironment) Terminate(kill bool) {
	if kill == true {
		de.client.RemoveContainer(docker.RemoveContainerOptions{
			ID:    de.container.ID,
			Force: true,
		})
		return
	}
	start := time.Now()
	de.inputWaiter.Close()
	de.outputWaiter.Close()
	de.client.StopContainer(de.container.ID, 0)
	de.client.WaitContainer(de.container.ID)
	finish := time.Now()
	log.Debugf("Container %s terminated in %f seconds.", de.shortID,
		finish.Sub(start).Seconds())
}

func (de *DockerEnvironment) Execute(request *messages.ExecutionRequest) ([]byte, []byte, error) {
	de.resetStreams()
	start := time.Now()
	execRequest := de.prepareRequest(request)
	de.encoder.Encode(execRequest)
	for {
		if de.stdout.Len() > 0 {
			break
		}
	}
	finish := time.Now()
	log.Infof("Docker container %s ran %s for %f secs.", de.shortID, request.Command, finish.Sub(start).Seconds())
	output, errors, err := de.parseResponse(execRequest.Executable)
	go func() {
		de.Terminate(false)
	}()
	return output, errors, err
}

func (de *DockerEnvironment) resetStreams() {
	de.stdin.Reset()
	de.stdout.Reset()
}

func (de *DockerEnvironment) connectStreams() error {
	outputWaiter, err := de.client.AttachToContainerNonBlocking(docker.AttachToContainerOptions{
		Container:    de.container.ID,
		OutputStream: de.stdout,
		Stream:       true,
		Stdout:       true,
		Success:      de.barrier,
	})
	if err != nil {
		return err
	}
	de.outputWaiter = outputWaiter
	// Wait for client to set up streams w/Docker
	<-de.barrier
	de.barrier <- struct{}{}
	inputWaiter, err := de.client.AttachToContainerNonBlocking(docker.AttachToContainerOptions{
		Container:   de.container.ID,
		InputStream: de.stdin,
		Stream:      true,
		Stdin:       true,
	})
	if err != nil {
		outputWaiter.Close()
		outputWaiter.Wait()
		de.outputWaiter = nil
		return err
	}
	de.inputWaiter = inputWaiter
	de.encoder = gob.NewEncoder(de.stdin)
	de.decoder = gob.NewDecoder(de.stdout)
	return err
}

func (de *DockerEnvironment) createContainerOptions() (*docker.CreateContainerOptions, error) {
	imageID := fmt.Sprintf("%s:%s", de.bundle.Docker.Image, de.bundle.Docker.Tag)
	containerMemory := de.relayConfig.Docker.ContainerMemory
	return &docker.CreateContainerOptions{
		Name: "",
		Config: &docker.Config{
			Image:      imageID,
			Memory:     int64(containerMemory * Megabyte),
			MemorySwap: 0,
			OpenStdin:  true,
			Labels: map[string]string{
				RelayCreatedDockerLabel: "yes",
			},
			Cmd: []string{"/operable/cogexec/bin/cogexec"},
		},
		HostConfig: &docker.HostConfig{
			VolumesFrom: []string{"cogexec"},
			Privileged:  false,
		},
	}, nil
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

func (de *DockerEnvironment) parseResponse(executable string) ([]byte, []byte, error) {
	resp := &exemsg.ExecCommandResponse{}
	if err := de.decoder.Decode(resp); err != nil {
		log.Errorf("Error decoding cogexec response: %s.", err)
		return EmptyResult, EmptyResult, err
	} else {
		log.Debugf("Docker container %s executed '%s' in %f secs.", de.shortID, executable, resp.Elapsed.Seconds())
	}
	return resp.Stdout, resp.Stderr, nil
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
