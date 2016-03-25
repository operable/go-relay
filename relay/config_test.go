package relay

import (
	"os"
	"testing"
)

const (
	fullConfig = `id: 2bba0d1f-a30c-45ec-87e6-e4c5d8c6104f
max_concurrent: 32
token: wubba
cog:
  id: 2bba0d1f-a30c-45ec-87e6-e4c5d8c6104a
  host: 127.0.0.1
  port: 1883
docker:
  use_env: false
  socket_path: /var/run/docker.sock
execution:
  cpu_shares: 16
  cpu_set: default
  env: ["TEST1=a", "TEST2=b"]
`
	usingDefaultsConfig = `id: 2bba0d1f-a30c-45ec-87e6-e4c5d8c6104f
token: wubba
cog:
  id: 2bba0d1f-a30c-45ec-87e6-e4c5d8c6104a
docker:
  use_env: false
execution:
  cpu_shares: 16
  cpu_set: default
  env: ["TEST1=a", "TEST2=b"]
`
)

func TestLoadConfig(t *testing.T) {
	relayID := "2bba0d1f-a30c-45ec-87e6-e4c5d8c6104f"
	cogID := "2bba0d1f-a30c-45ec-87e6-e4c5d8c6104a"
	config, err := ParseConfig([]byte(fullConfig))
	if err != nil {
		t.Logf("UseEnv: %s", config.Docker.UseEnv)
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

	// Validate Cog config
	cog := config.Cog
	if cog.ID != cogID {
		t.Errorf("Expected Cog ID '%s'", cogID)
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
	if docker.SocketPath != "/var/run/docker.sock" {
		t.Errorf("Expected default docker/socket_path of '/var/run/docker.sock': %s", docker.SocketPath)
	}
}

func TestApplyEnvVars(t *testing.T) {
	os.Setenv("RELAY_MAX_CONCURRENT", "8")
	os.Setenv("RELAY_COG_PORT", "1880")
	os.Setenv("RELAY_DOCKER_SOCKET_PATH", "/foo/bar/baz.sock")
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
	if config.Docker.SocketPath != "/foo/bar/baz.sock" {
		t.Errorf("Expected docker/socket_path to be /foo/bar/baz/sock: %s", config.Docker.SocketPath)
	}
}
