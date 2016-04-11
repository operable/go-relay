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
  env: ["TEST1=a", "TEST2=b"]
`
	disabledDockerConfig = `id: 2bba0d1f-a30c-45ec-87e6-e4c5d8c6104f
version: 1
enabled_engines: native
cog:
  token: wubba
execution:
  env: ["TEST1=a", "TEST2=b"]
`
)

func TestLoadConfig(t *testing.T) {
	os.Clearenv()
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
	if config.Execution != nil {
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
	os.Clearenv()
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
	if docker.RegistryHost != "index.docker.io" {
		t.Errorf("Expected default docker/registry_host of 'index.docker.io': %s", docker.RegistryHost)
	}
}

func TestApplyEnvVars(t *testing.T) {
	os.Clearenv()
	os.Setenv("RELAY_MAX_CONCURRENT", "8")
	os.Setenv("RELAY_COG_PORT", "1880")
	os.Setenv("RELAY_COG_REFRESH_INTERVAL", "60s")
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
	if config.NativeEnabled() != true {
		t.Error("Expected NativeEnabled() to return true")
	}
	if config.DockerEnabled() != true {
		t.Error("Expected DockerEnabled() to return true")
	}
	if config.Cog.RefreshInterval != "60s" {
		t.Errorf("Expected cog/refresh_interval to be '60s': %s", config.Cog.RefreshInterval)
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

func TestBadExecutionEngineName(t *testing.T) {
	os.Clearenv()
	os.Setenv("RELAY_ENABLED_ENGINES", "kvm")
	config, err := ParseConfig([]byte(fullConfig))
	if err != nil {
		t.Fatal(err)
	}
	if config.DockerEnabled() == true {
		t.Error("Expected DockerEnabled() to return false")
	}
	if config.NativeEnabled() == true {
		t.Error("Expected NativeEnabled() to return false")
	}
}

func TestDisabledDocker(t *testing.T) {
	config, err := ParseConfig([]byte(disabledDockerConfig))
	if err != nil {
		t.Fatal(err)
	}
	if config.DockerEnabled() == true {
		t.Error("Expected DockerEnabled() to return false")
	}
	if config.Docker != nil {
		t.Errorf("Expected missing Docker config: %+v", config.Docker)
	}
}
