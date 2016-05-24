package engines

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	log "github.com/Sirupsen/logrus"
	"github.com/fsouza/go-dockerclient"
	"github.com/operable/go-relay/relay/config"
	"github.com/operable/go-relay/relay/messages"
	"strings"
	"time"
)

var errorDockerDisabled = errors.New("Docker engine is disabled")

var relayCreatedLabel = "io.operable.cog.relay.created"
var relayCreatedFilter = "io.operable.cog.relay.created=yes"

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
	if relayConfig.DockerEnabled() == false {
		return nil, errorDockerDisabled
	}
	dockerConfig := *relayConfig.Docker
	client, err := newClient(dockerConfig)
	if err != nil {
		return nil, err
	}
	return &DockerEngine{
		client:      client,
		relayConfig: relayConfig,
		config:      dockerConfig,
	}, nil
}

// IsAvailable returns true/false if a Docker image is found
func (de *DockerEngine) IsAvailable(name string, meta string) (bool, error) {
	log.Debugf("Retrieving latest Docker image for %s:%s from upstream Docker registry.", name, meta)
	beforeID, _ := de.IDForName(name, meta)
	pullErr := de.client.PullImage(docker.PullImageOptions{
		Repository: name,
		Tag:        meta,
	}, de.makeAuthConfig())
	if pullErr != nil {
		log.Errorf("Error ocurred when pulling image %s: %s.", name, pullErr)
		image, inspectErr := de.client.InspectImage(name)
		if inspectErr != nil || image == nil {
			log.Errorf("Unable to find Docker image %s locally or in remote registry.", name)
			return false, pullErr
		}
		log.Infof("Retrieving Docker image %s from remote registry failed. Falling back to local copy, if it exists.", name)
		return image != nil, nil
	}
	afterID, err := de.IDForName(name, meta)
	if err != nil {
		return false, err
	}
	if beforeID == "" {
		log.Infof("Retrieved Docker image %s for %s:%s.", shortImageID(afterID), name, meta)
		return true, nil
	}
	if beforeID != afterID {
		if removeErr := de.client.RemoveImageExtended(beforeID,
			docker.RemoveImageOptions{Force: true}); removeErr != nil {
			log.Errorf("Failed to remove old Docker image %s: %s.", shortImageID(beforeID), removeErr)
		} else {
			log.Infof("Replaced obsolete Docker image %s with %s.", shortImageID(beforeID), shortImageID(afterID))
		}
	} else {
		log.Infof("Docker image %s for %s:%s is up to date.", shortImageID(beforeID), name, meta)
	}
	return true, nil
}

// Execute a command inside a Docker container
func (de *DockerEngine) Execute(request *messages.ExecutionRequest, bundle *config.Bundle) ([]byte, []byte, error) {
	createOptions, err := de.createContainerOptions(request, bundle)
	if err != nil {
		return emptyResult, emptyResult, err
	}
	container, err := de.client.CreateContainer(*createOptions)
	if err != nil {
		return emptyResult, emptyResult, err
	}
	containerID := shortContainerID(container.ID)
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
	log.Infof("Docker container %s ran %s for %f secs.", containerID, request.Command, finish.Sub(start).Seconds())
	return de.stdout.Bytes(), de.stderr.Bytes(), nil
}

// VerifyDockerConfig sanity checks Docker configuration and ensures Relay
// can talk to Docker.
func VerifyDockerConfig(dockerConfig *config.DockerInfo) error {
	client, err := newClient(*dockerConfig)
	if err != nil {
		return err
	}
	if _, err := client.Info(); err != nil {
		return err
	}
	return verifyCredentials(client, dockerConfig)
}

// IDForName returns the image ID for a given image name
func (de *DockerEngine) IDForName(name string, meta string) (string, error) {
	image, err := de.client.InspectImage(fmt.Sprintf("%s:%s", name, meta))
	if err != nil {
		return "", err
	}
	return image.ID, nil
}

// Clean removes exited containers
func (de *DockerEngine) Clean() int {
	containers, err := de.client.ListContainers(docker.ListContainersOptions{
		Filters: map[string][]string{
			"status": []string{"exited"},
			"label":  []string{relayCreatedFilter},
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
			log.Errorf("Error removing Docker container %s: %s.", shortContainerID(container.ID), err)
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

func (de *DockerEngine) createContainerOptions(request *messages.ExecutionRequest, bundle *config.Bundle) (*docker.CreateContainerOptions, error) {
	imageID, err := de.IDForName(bundle.Docker.Image, bundle.Docker.Tag)
	if err != nil {
		return nil, err
	}
	command := request.CommandName()
	return &docker.CreateContainerOptions{
		Name: "",
		Config: &docker.Config{
			Image:      imageID,
			Env:        BuildEnvironment(*request, de.relayConfig),
			Memory:     64 * 1024 * 1024, // 64MB
			MemorySwap: 0,
			StdinOnce:  true,
			OpenStdin:  true,
			Labels: map[string]string{
				relayCreatedLabel: "yes",
			},
			Cmd: []string{bundle.Commands[command].Executable},
		},
		HostConfig: &docker.HostConfig{
			Privileged: false,
		},
	}, nil
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
	if dockerConfig.RegistryUser == "" || dockerConfig.RegistryPassword == "" ||
		dockerConfig.RegistryEmail == "" {
		log.Info("No Docker registry credentials found or credentials are incomplete. Skipping auth check.")
		return nil
	}
	log.Info("Verifying Docker registry credentials.")
	authConf := docker.AuthConfiguration{
		Username:      dockerConfig.RegistryUser,
		Password:      dockerConfig.RegistryPassword,
		Email:         dockerConfig.RegistryEmail,
		ServerAddress: dockerConfig.RegistryHost,
	}
	_, err := client.AuthCheck(&authConf)
	return err
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

func shortContainerID(containerID string) string {
	idEnd := len(containerID)
	idStart := idEnd - 12
	return containerID[idStart:idEnd]
}

func shortImageID(imageID string) string {
	chunks := strings.Split(imageID, ":")
	return chunks[1][:11]
}
