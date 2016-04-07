package engines

import (
	"bytes"
	"encoding/json"
	log "github.com/Sirupsen/logrus"
	"github.com/fsouza/go-dockerclient"
	"github.com/operable/go-relay/relay/config"
	"github.com/operable/go-relay/relay/messages"
)

// DockerEngine is responsible for managing execution of
// Docker bundled commands.
type DockerEngine struct {
	client *docker.Client
	config *config.DockerInfo
	stdout []byte
	stderr []byte
}

// NewDockerEngine makes a new DockerEngine instance
func NewDockerEngine(dockerConfig *config.DockerInfo) (Engine, error) {
	if dockerConfig == nil {
		return nil, nil
	}
	client, err := newClient(dockerConfig)
	if err != nil {
		return nil, err
	}
	return &DockerEngine{
		client: client,
		config: dockerConfig,
	}, nil
}

// IsAvailable returns true/false if a Docker image is found
func (de *DockerEngine) IsAvailable(name string, meta string) (bool, error) {
	err := de.client.PullImage(docker.PullImageOptions{
		Repository: name,
		Tag:        meta,
	}, docker.AuthConfiguration{})
	return err == nil, err
}

// Execute a command inside a Docker container
func (de *DockerEngine) Execute(request *messages.ExecutionRequest, bundle *config.Bundle) ([]byte, []byte, error) {
	emptyResult := []byte{}
	container, err := de.client.CreateContainer(de.createContainerOptions(request.CommandName(), bundle))
	if err != nil {
		return emptyResult, emptyResult, err
	}
	log.Infof("Container %s created for command %s", container.ID, request.Command)
	rendezvous := make(chan int)
	go func() {
		input, _ := json.Marshal(request.CogEnv)
		de.writeToStdin(container.ID, input, rendezvous)
	}()
	go func() {
		de.readOutput(container.ID, rendezvous)
	}()
	for i := 0; i < 2; i++ {
		<-rendezvous
	}
	err = de.client.StartContainer(container.ID, nil)
	if err != nil {
		return emptyResult, emptyResult, err
	}
	log.Infof("Container %s started.", container.ID)
	de.client.WaitContainer(container.ID)
	de.client.StopContainer(container.ID, 5)
	log.Info("Container %s finished.", container.ID)
	<-rendezvous
	go func() {
		de.client.RemoveContainer(docker.RemoveContainerOptions{
			ID:            container.ID,
			RemoveVolumes: true,
			Force:         true,
		})
	}()
	return de.stdout, de.stderr, nil
}

// VerifyDockerConfig sanity checks Docker configuration and ensures Relay
// can talk to Docker.
func VerifyDockerConfig(dockerConfig *config.DockerInfo) error {
	client, err := newClient(dockerConfig)
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

func (de *DockerEngine) writeToStdin(containerID string, input []byte, rendezvous chan int) {
	client, _ := newClient(de.config)
	rendezvous <- 1
	client.AttachToContainer(docker.AttachToContainerOptions{
		Container:   containerID,
		InputStream: bytes.NewBuffer(input),
		Stdin:       true,
		Stream:      true,
	})
}

func (de *DockerEngine) readOutput(containerID string, rendezvous chan int) {
	var (
		output bytes.Buffer
		errors bytes.Buffer
	)
	client, _ := newClient(de.config)
	rendezvous <- 1
	client.AttachToContainer(docker.AttachToContainerOptions{
		Container:    containerID,
		Stdout:       true,
		Stderr:       true,
		Stream:       true,
		OutputStream: &output,
		ErrorStream:  &errors,
	})
	de.stdout = output.Bytes()
	de.stderr = errors.Bytes()
	rendezvous <- 1
}

func (de *DockerEngine) createContainerOptions(command string, bundle *config.Bundle) docker.CreateContainerOptions {
	return docker.CreateContainerOptions{
		Name: "",
		Config: &docker.Config{
			Image:      bundle.Docker.ID,
			Env:        []string{},
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
