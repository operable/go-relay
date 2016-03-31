package docker

import (
	"github.com/fsouza/go-dockerclient"
	"github.com/operable/go-relay/relay"
)

func VerifyConfig(config *relay.DockerInfo) error {
	if config.UseEnv {
		client, err := docker.NewClientFromEnv()
		if err != nil {
			return err
		}
		return verifyCredentials(client, config)
	}
	client, err := docker.NewClient(config.SocketPath)
	if err != nil {
		return err
	}
	return verifyCredentials(client, config)
}

func verifyCredentials(client *docker.Client, config *relay.DockerInfo) error {
	authConf := docker.AuthConfiguration{
		Username:      config.RegistryUser,
		Password:      config.RegistryPassword,
		ServerAddress: config.RegistryHost,
	}
	return client.AuthCheck(&authConf)
}
