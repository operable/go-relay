package engines

import (
	"bytes"
	"encoding/json"
	"errors"
	log "github.com/Sirupsen/logrus"
	"github.com/fsouza/go-dockerclient"
	"github.com/operable/go-relay/relay/config"
	"github.com/operable/go-relay/relay/messages"
	"time"
)

var dockerDisabledError = errors.New("Docker engine is disabled")

// DockerEngine is responsible for managing execution of
// Docker bundled commands.
type DockerEngine struct {
	client      *docker.Client
	relayConfig config.Config
	config      config.DockerInfo
	stdout      bytes.Buffer
	stderr      bytes.Buffer
}

// NewDockerEngine makes a new DockerEngine instance
func NewDockerEngine(relayConfig config.Config) (Engine, error) {
	dockerConfig := relayConfig.Docker
	if dockerConfig == nil {
		return nil, dockerDisabledError
	}
	client, err := newClient(*dockerConfig)
	if err != nil {
		return nil, err
	}
	return &DockerEngine{
		client:      client,
		relayConfig: relayConfig,
		config:      *dockerConfig,
	}, nil
}

// IsAvailable returns true/false if a Docker image is found
func (de *DockerEngine) IsAvailable(name string, meta string) (bool, error) {
	pullErr := de.client.PullImage(docker.PullImageOptions{
		Repository: name,
		Tag:        meta,
	}, de.makeAuthConfig())
	if pullErr != nil {
		image, inspectErr := de.client.InspectImage(name)
		if inspectErr != nil || image == nil {
			log.Errorf("Unable to find image %s locally or in remote registry.", name)
			return false, pullErr
		} else {
			log.Infof("Retrieving image %s from remote registry failed. Falling back to local copy.", name)
		}
		return image != nil, nil
	}
	return true, nil
}

// Execute a command inside a Docker container
func (de *DockerEngine) Execute(request *messages.ExecutionRequest, bundle *config.Bundle) ([]byte, []byte, error) {
	container, err := de.client.CreateContainer(de.createContainerOptions(request, bundle))
	if err != nil {
		return emptyResult, emptyResult, err
	}
	containerID := shortID(container.ID)
	input, _ := json.Marshal(request.CogEnv)
	inputWaiter, err := de.attachInputWriter(container.ID, input)
	if err != nil {
		de.removeContainer(container.ID)
		return emptyResult, emptyResult, err
	}
	outputWaiter, err := de.attachOutputReader(container.ID)
	if err != nil {
		inputWaiter.Close()
		de.removeContainer(container.ID)
		return emptyResult, emptyResult, err
	}
	start := time.Now()
	err = de.client.StartContainer(container.ID, nil)
	if err != nil {
		inputWaiter.Close()
		outputWaiter.Close()
		return emptyResult, emptyResult, err
	}
	de.client.WaitContainer(container.ID)
	finish := time.Now()
	log.Infof("Container %s ran %s for %f secs.", containerID, request.Command, finish.Sub(start).Seconds())
	return de.stdout.Bytes(), de.stderr.Bytes(), nil
}

// VerifyDockerConfig sanity checks Docker configuration and ensures Relay
// can talk to Docker.
func VerifyDockerConfig(dockerConfig *config.DockerInfo) error {
	client, err := newClient(*dockerConfig)
	if err != nil {
		return err
	}
	return verifyCredentials(client, dockerConfig)
}

// IDForName returns the image ID for a given image name
func (de *DockerEngine) IDForName(name string) (string, error) {
	image, err := de.client.InspectImage(name)
	if err != nil {
		return "", err
	}
	return image.ID, nil
}

func (de *DockerEngine) Clean() int {
	containers, err := de.client.ListContainers(docker.ListContainersOptions{
		Filters: map[string][]string{
			"status": []string{"exited"},
		},
	})
	if err != nil {
		log.Errorf("Listing dead Docker containers failed: %s.", err)
		return 0
	}
	count := 0
	for _, container := range containers {
		err = de.removeContainer(container.ID)
		if err != nil {
			log.Errorf("Error removing container %s: %s.", shortID(container.ID), err)
		} else {
			count++
		}
	}
	return count
}

func (de *DockerEngine) attachInputWriter(containerID string, input []byte) (docker.CloseWaiter, error) {
	client, _ := newClient(de.config)
	return client.AttachToContainerNonBlocking(docker.AttachToContainerOptions{
		Container:   containerID,
		InputStream: bytes.NewBuffer(input),
		Stdin:       true,
		Stream:      true,
	})
}

func (de *DockerEngine) attachOutputReader(containerID string) (docker.CloseWaiter, error) {
	client, _ := newClient(de.config)
	return client.AttachToContainerNonBlocking(docker.AttachToContainerOptions{
		Container:    containerID,
		Stdout:       true,
		Stderr:       true,
		Stream:       true,
		OutputStream: &de.stdout,
		ErrorStream:  &de.stderr,
	})
}

func (de *DockerEngine) createContainerOptions(request *messages.ExecutionRequest, bundle *config.Bundle) docker.CreateContainerOptions {
	command := request.CommandName()
	return docker.CreateContainerOptions{
		Name: "",
		Config: &docker.Config{
			Image:      bundle.Docker.ID,
			Env:        BuildEnvironment(*request, de.relayConfig),
			Memory:     64 * 1024 * 1024, // 64MB
			MemorySwap: 0,
			StdinOnce:  true,
			OpenStdin:  true,
			Cmd:        []string{bundle.Commands[command].Executable},
		},
		HostConfig: &docker.HostConfig{
			Privileged: false,
		},
	}
}

func (de *DockerEngine) removeContainer(id string) error {
	return de.client.RemoveContainer(docker.RemoveContainerOptions{
		ID:            id,
		RemoveVolumes: true,
		Force:         true,
	})
}

func (de *DockerEngine) makeAuthConfig() docker.AuthConfiguration {
	return docker.AuthConfiguration{
		ServerAddress: de.config.RegistryHost,
		Username:      de.config.RegistryUser,
		Password:      de.config.RegistryPassword,
	}
}

func verifyCredentials(client *docker.Client, dockerConfig *config.DockerInfo) error {
	if dockerConfig.RegistryUser == "" || dockerConfig.RegistryPassword == "" {
		log.Info("No Docker registry credentials found. Skipping auth check.")
		return nil
	}
	log.Info("Verifying Docker registry credentials.")
	authConf := docker.AuthConfiguration{
		Username:      dockerConfig.RegistryUser,
		Password:      dockerConfig.RegistryPassword,
		ServerAddress: dockerConfig.RegistryHost,
	}
	return client.AuthCheck(&authConf)
}

func newClient(dockerConfig config.DockerInfo) (*docker.Client, error) {
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

func shortID(containerID string) string {
	idEnd := len(containerID)
	idStart := idEnd - 10
	return containerID[idStart:idEnd]
}
