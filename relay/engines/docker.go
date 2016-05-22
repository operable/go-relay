package engines

import (
	"fmt"
	log "github.com/Sirupsen/logrus"
	"github.com/fsouza/go-dockerclient"
	"github.com/operable/go-relay/relay/config"
	"github.com/operable/go-relay/relay/engines/exec"
	"strings"
)

var relayCreatedFilter = "io.operable.cog.relay.created=yes"

// DockerEngine is responsible for managing execution of
// Docker bundled commands.
type DockerEngine struct {
	client      *docker.Client
	relayConfig *config.Config
	config      config.DockerInfo
}

// NewDockerEngine makes a new DockerEngine instance
func NewDockerEngine(relayConfig *config.Config) (Engine, error) {
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

func (de *DockerEngine) Init() error {
	err := de.client.PullImage(docker.PullImageOptions{
		Repository: "operable/cogexec",
	}, de.makeAuthConfig())
	if err != nil {
		log.Errorf("Failed to pull required operable/cogexec image: %s.", err)
		return err
	}
	log.Info("Updated operable/cogexec image.")
	return de.createCogexec()
}

// IsAvailable returns true/false if a Docker image is found
func (de *DockerEngine) IsAvailable(name string, meta string) (bool, error) {
	imageName := fmt.Sprintf("%s:%s", name, meta)
	log.Debugf("Retrieving latest Docker image for %s from upstream Docker registry.", imageName)
	beforeID, _ := de.IDForName(name, meta)
	pullErr := de.client.PullImage(docker.PullImageOptions{
		Repository: name,
		Tag:        meta,
	}, de.makeAuthConfig())
	if pullErr != nil {
		log.Errorf("Error ocurred when pulling image %s: %s.", imageName, pullErr)
		image, inspectErr := de.client.InspectImage(imageName)
		if inspectErr != nil || image == nil {
			log.Errorf("Unable to find Docker image %s locally or in remote registry.", imageName)
			return false, pullErr
		}
		log.Infof("Retrieving Docker image %s from remote registry failed. Falling back to local copy, if it exists.", imageName)
		return image != nil, nil
	}
	afterID, err := de.IDForName(name, meta)
	if err != nil {
		return false, err
	}
	if beforeID == "" {
		log.Infof("Retrieved Docker image %s for %s.", shortImageID(afterID), imageName)
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
		log.Infof("Docker image %s for %s is up to date.", shortImageID(beforeID), imageName)
	}
	return true, nil
}

// NewEnvironment is required by the engines.Engine interface
func (de *DockerEngine) NewEnvironment(bundle *config.Bundle) (exec.Environment, error) {
	return exec.NewDockerEnvironment(de.relayConfig, bundle)
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

func (de *DockerEngine) createCogexec() error {
	// Just in case
	de.client.RemoveContainer(docker.RemoveContainerOptions{
		ID:    "cogexec",
		Force: true,
	})
	_, err := de.client.CreateContainer(docker.CreateContainerOptions{
		Name: "cogexec",
		Config: &docker.Config{
			Cmd:   []string{"/bin/date"},
			Image: "operable/cogexec",
		},
	})
	if err != nil {
		log.Errorf("Creation of required cogexec container failed: %s.", err)
		return err
	}
	log.Info("Created required cogexec container.")
	return nil
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
