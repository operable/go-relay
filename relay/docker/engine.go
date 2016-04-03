package docker

import (
	log "github.com/Sirupsen/logrus"
	"github.com/fsouza/go-dockerclient"
	"github.com/operable/go-relay/relay/config"
)

// Engine is responsible for managing execution of
// Docker bundled commands.
type Engine struct {
	client *docker.Client
	config *config.DockerInfo
}

// NewEngine makes a new Engine instance
func NewEngine(dockerConfig *config.DockerInfo) (*Engine, error) {
	client, err := newClient(dockerConfig)
	if err != nil {
		return nil, err
	}
	return &Engine{
		client: client,
		config: dockerConfig,
	}, nil
}

// VerifyConfig sanity checks Docker configuration and ensures Relay
// can talk to Docker.
func VerifyConfig(dockerConfig *config.DockerInfo) error {
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
