package engines

import (
	log "github.com/Sirupsen/logrus"
	"github.com/fsouza/go-dockerclient"
	"github.com/operable/go-relay/relay/config"
)

// DockerEngine is responsible for managing execution of
// Docker bundled commands.
type DockerEngine struct {
	client *docker.Client
	config *config.DockerInfo
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

func (de *DockerEngine) IsAvailable(interface{}) (bool, error) {
	return false, nil
}

func (de *DockerEngine) Execute(interface{}) ([]byte, error) {
	return []byte{0}, nil
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
