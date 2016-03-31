package docker

import (
	log "github.com/Sirupsen/logrus"
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
	if config.RegistryUser == "" || config.RegistryPassword == "" {
		log.Info("No Docker registry credentials found. Skipping auth check.")
		return nil
	}
	log.Info("Verifying Docker registry credentials.")
	authConf := docker.AuthConfiguration{
		Username:      config.RegistryUser,
		Password:      config.RegistryPassword,
		ServerAddress: config.RegistryHost,
	}
	return client.AuthCheck(&authConf)
}
