package config

import (
	"os"
	"testing"
)

const (
	fullConfig = `id: 2bba0d1f-a30c-45ec-87e6-e4c5d8c6104f
max_concurrent: 32
version: 1
cog:
  host: 127.0.0.1
  port: 1883
  token: wubba
docker:
  use_env: false
  socket_path: unix:///var/run/docker.sock
  registry_user: testy
  registry_password: test123
execution:
  cpu_shares: 16
  cpu_set: default
  env: ["TEST1=a", "TEST2=b"]
`
	usingDefaultsConfig = `id: 2bba0d1f-a30c-45ec-87e6-e4c5d8c6104f
version: 1
cog:
  token: wubba
docker:
  use_env: false
  registry_user: testy
  registry_password: test123
execution:
  cpu_shares: 16
  cpu_set: default
  env: ["TEST1=a", "TEST2=b"]
`
	disabledDockerConfig = `id: 2bba0d1f-a30c-45ec-87e6-e4c5d8c6104f
version: 1
disable_docker: true
cog:
  token: wubba
execution:
  cpu_shares: 16
  cpu_set: default
  env: ["TEST1=a", "TEST2=b"]
`
)

func TestLoadConfig(t *testing.T) {
	relayID := "2bba0d1f-a30c-45ec-87e6-e4c5d8c6104f"
	config, err := ParseConfig([]byte(fullConfig))
	if err != nil {
		t.Fatal(err)
	}
	// Validate top level config
	if config.Cog == nil {
		t.Errorf("Expected non-nil Cog config")
	}
	if config.Docker == nil {
		t.Errorf("Expected non-nil Docker config")
	}
	if config.Execution == nil {
		t.Errorf("Expected non-nil Execution config")
	}
	if config.ID != relayID {
		t.Errorf("Expected Relay ID '%s'", relayID)
	}
	if config.MaxConcurrent != 32 {
		t.Errorf("Expected MaxConcurent 32")
	}
}

func TestApplyConfigDefaults(t *testing.T) {
	config, err := ParseConfig([]byte(usingDefaultsConfig))
	if err != nil {
		t.Fatal(err)
	}
	if config.MaxConcurrent != 16 {
		t.Errorf("Expected default max_concurrent of 16: %d", config.MaxConcurrent)
	}
	cog := config.Cog
	if cog.Host != "127.0.0.1" {
		t.Errorf("Expected default cog/host of '127.0.0.1': %s", cog.Host)
	}
	if cog.Port != 1883 {
		t.Errorf("Expected default cog/port of 1883: %d", cog.Port)
	}

	docker := config.Docker
	if docker.SocketPath != "unix:///var/run/docker.sock" {
		t.Errorf("Expected default docker/socket_path of 'unix:///var/run/docker.sock': %s", docker.SocketPath)
	}
	if docker.RegistryHost != "hub.docker.com" {
		t.Errorf("Expected default docker/registry_host of 'hub.docker.com': %s", docker.RegistryHost)
	}
}

func TestApplyEnvVars(t *testing.T) {
	os.Setenv("RELAY_MAX_CONCURRENT", "8")
	os.Setenv("RELAY_COG_PORT", "1880")
	os.Setenv("RELAY_DOCKER_SOCKET_PATH", "unix:///foo/bar/baz.sock")
	os.Setenv("RELAY_DOCKER_REGISTRY_USER", "testuser")
	os.Setenv("RELAY_DOCKER_REGISTRY_PASSWORD", "testy")
	config, err := ParseConfig([]byte(fullConfig))

	if err != nil {
		t.Fatal(err)
	}
	if config.MaxConcurrent != 8 {
		t.Errorf("Expected max_concurrent to be 8: %d", config.MaxConcurrent)
	}
	if config.Cog.Port != 1880 {
		t.Errorf("Expected cog/port to be 1880: %d", config.Cog.Port)
	}
	if config.Docker.SocketPath != "unix:///foo/bar/baz.sock" {
		t.Errorf("Expected docker/socket_path to be unix:///foo/bar/baz/sock: %s", config.Docker.SocketPath)
	}
	if config.Docker.RegistryUser != "testuser" {
		t.Errorf("Expected docker/registry_user to be 'testuser': %s", config.Docker.RegistryUser)
	}
	if config.Docker.RegistryPassword != "testy" {
		t.Errorf("Expected docker/registry_password to be 'testy': %s", config.Docker.RegistryPassword)
	}
}

func TestDisabledDocker(t *testing.T) {
	config, err := ParseConfig([]byte(disabledDockerConfig))
	if err != nil {
		t.Fatal(err)
	}
	if config.Docker != nil {
		t.Errorf("Expected missing Docker config: %+v", config.Docker)
	}
}
