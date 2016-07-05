package config

import (
	"errors"
	"time"
)

var errorBadCleanInterval = errors.New("Error parsing docker/clean_interval")

// DockerInfo contains information required to interact with dockerd and external Docker registries
type DockerInfo struct {
	UseEnv               bool   `yaml:"use_env" env:"RELAY_DOCKER_USE_ENV" valid:"-" default:"false"`
	SocketPath           string `yaml:"socket_path" env:"RELAY_DOCKER_SOCKET_PATH" valid:"dockersocket,required" default:"unix:///var/run/docker.sock"`
	ContainerMemory      int    `yaml:"container_memory" env:"RELAY_DOCKER_CONTAINER_MEMORY" valid:"required" default:"16"`
	CleanInterval        string `yaml:"clean_interval" env:"RELAY_DOCKER_CLEAN_INTERVAL" valid:"required" default:"5m"`
	CommandDriverVersion string `yaml:"command_driver_version" env:"RELAY_DOCKER_CIRCUIT_DRIVER_VERSION" valid:"required" default:"0.9"`
	RegistryHost         string `yaml:"registry_host" env:"RELAY_DOCKER_REGISTRY_HOST" valid:"host,required" default:"index.docker.io"`
	RegistryUser         string `yaml:"registry_user" env:"RELAY_DOCKER_REGISTRY_USER" valid:"-"`
	RegistryEmail        string `yaml:"registry_email" env:"RELAY_DOCKER_REGISTRY_EMAIL" valid:"-"`
	RegistryPassword     string `yaml:"registry_password" env:"RELAY_DOCKER_REGISTRY_PASSWORD" valid:"-"`
}

// CleanDuration returns CleanInterval as a time.Duration
func (di *DockerInfo) CleanDuration() time.Duration {
	duration, err := time.ParseDuration(di.CleanInterval)
	if err != nil {
		panic(errorBadCleanInterval)
	}
	return duration
}
