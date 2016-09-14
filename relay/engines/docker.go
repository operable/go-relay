package engines

import (
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	log "github.com/Sirupsen/logrus"
	"github.com/docker/engine-api/client"
	"github.com/docker/engine-api/types"
	"github.com/docker/engine-api/types/container"
	"github.com/docker/engine-api/types/filters"
	"github.com/operable/circuit"
	"github.com/operable/go-relay/relay/config"
	"golang.org/x/net/context"
	"io/ioutil"
	"os"
	"strings"
)

const (
	megabyte = 1024 * 1024
)

var relayCreatedLabel = "io.operable.cog.relay.create"
var errorDriverImageUnavailable = errors.New("Command driver image is unavailable")

// DockerEngine is responsible for managing execution of
// Docker bundled commands.
type DockerEngine struct {
	client      *client.Client
	relayConfig *config.Config
	config      config.DockerInfo
	auth        string
	cache       *envCache
}

// NewDockerEngine makes a new DockerEngine instance
func NewDockerEngine(relayConfig *config.Config, cache *envCache) (Engine, error) {
	dockerConfig := *relayConfig.Docker
	return &DockerEngine{
		client:      nil,
		relayConfig: relayConfig,
		config:      dockerConfig,
		cache:       cache,
	}, nil
}

// Init is required by the engines.Engine interface
func (de *DockerEngine) Init() error {
	return de.createCircuitDriver()
}

// IsAvailable returns true/false if a Docker image is found
func (de *DockerEngine) IsAvailable(name string, meta string) (bool, error) {
	err := de.ensureConnected()
	if err != nil {
		return false, err
	}

	if de.needsUpdate(name, meta) == false {
		return true, nil
	}
	fullName := fmt.Sprintf("%s:%s", name, meta)
	err = de.attemptAuth()
	if err != nil {
		return false, err
	}
	log.Debugf("Retrieving %s from upstream Docker registry.", fullName)
	beforeID, _ := de.IDForName(name, meta)
	pullErr := de.pullImage(fullName)
	if pullErr != nil {
		log.Errorf("Error ocurred pulling image %s: %s.", name, pullErr)
		return false, pullErr
	}
	log.Debugf("Retrieved %s from upstream Docker registry.", fullName)
	afterID, err := de.IDForName(name, meta)
	if err != nil {
		log.Errorf("Image pull completed but no image available for name %s : %s", fullName, err)
		return false, err
	}
	de.removeOldImage(beforeID, afterID, fullName)
	return true, nil
}

// NewEnvironment is required by the engines.Engine interface
func (de *DockerEngine) NewEnvironment(pipelineID string, bundle *config.Bundle) (circuit.Environment, error) {
	key := makeKey(pipelineID, bundle)
	if cached := de.cache.get(key); cached != nil {
		return cached, nil
	}
	log.Debugf("Creating environment %s", key)
	return de.newEnvironment(bundle)
}

// ReleaseEnvironment is required by the engines.Engine interface
func (de *DockerEngine) ReleaseEnvironment(pipelineID string, bundle *config.Bundle, env circuit.Environment) {
	key := makeKey(pipelineID, bundle)
	if de.cache.put(key, env) == false {
		env.Shutdown()
	}
}

// IDForName returns the image ID for a given image name
func (de *DockerEngine) IDForName(name string, meta string) (string, error) {
	err := de.ensureConnected()
	if err != nil {
		return "", err
	}
	image, _, err := de.client.ImageInspectWithRaw(context.Background(), fmt.Sprintf("%s:%s", name, meta), false)
	if err != nil {
		return "", err
	}
	return image.ID, nil
}

// Clean removes exited containers
func (de *DockerEngine) Clean() int {
	err := de.ensureConnected()
	if err != nil {
		return 0
	}
	count := 0
	for _, env := range de.cache.getOld() {
		if env.Shutdown() == nil {
			count++
		}
	}
	args := filters.NewArgs()
	args.Add("status", "exited")
	args.Add("label", fmt.Sprintf("%s=yes", relayCreatedLabel))
	containers, err := de.client.ContainerList(context.Background(),
		types.ContainerListOptions{
			Filter: args,
		})
	if err != nil {
		log.Errorf("Listing dead Docker containers failed: %s.", err)
		return 0
	}
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
	return de.client.ContainerRemove(context.Background(), id, types.ContainerRemoveOptions{
		RemoveVolumes: true,
		Force:         true,
	})
}

func (de *DockerEngine) makeAuthConfig() *types.AuthConfig {
	if de.config.RegistryUser == "" || de.config.RegistryPassword == "" || de.config.RegistryEmail == "" {
		return nil
	}
	return &types.AuthConfig{
		ServerAddress: de.config.RegistryHost,
		Username:      de.config.RegistryUser,
		Password:      de.config.RegistryPassword,
		Email:         de.config.RegistryEmail,
	}
}

func (de *DockerEngine) createCircuitDriver() error {
	err := de.ensureConnected()
	if err != nil {
		return err
	}
	// Just in case
	de.client.ContainerRemove(context.Background(), "cog-circuit-driver", types.ContainerRemoveOptions{
		RemoveVolumes: true,
		Force:         true,
	})
	avail, err := de.IsAvailable("operable/circuit-driver", de.config.CommandDriverVersion)
	if err != nil {
		return err
	}
	if avail == false {
		return errorDriverImageUnavailable
	}

	hostConfig := container.HostConfig{
		Privileged: false,
	}
	fullName := fmt.Sprintf("operable/circuit-driver:%s", de.config.CommandDriverVersion)
	hostConfig.Memory = int64(4 * megabyte)
	config := container.Config{
		Image:     fullName,
		Cmd:       []string{"/bin/date"},
		OpenStdin: false,
		StdinOnce: false,
		Env:       []string{},
		Labels: map[string]string{
			relayCreatedLabel: "yes",
		},
	}
	_, err = de.client.ContainerCreate(context.Background(), &config, &hostConfig, nil, "cog-circuit-driver")
	if err != nil {
		log.Errorf("Creation of required command driver container failed: %s.", err)
		return err
	}
	log.Info("Created required command driver container.")
	return nil
}

func (de *DockerEngine) attemptAuth() error {
	if de.auth == "" {
		authConfig := de.makeAuthConfig()
		if authConfig == nil {
			return nil
		}
		_, err := de.client.RegistryLogin(context.Background(), *authConfig)
		if err != nil {
			log.Errorf("Authenticating to Docker registry failed: %s.", err)
			return err
		}
		jsonAuth, err := json.Marshal(authConfig)
		if err != nil {
			return err
		}
		de.auth = base64.StdEncoding.EncodeToString(jsonAuth)
	}
	return nil
}

func (de *DockerEngine) developerModeRefresh(bundle *config.Bundle) error {
	if de.relayConfig.DevMode == true {
		fullName := fmt.Sprintf("%s:%s", bundle.Docker.Image, bundle.Docker.Tag)
		log.Warnf("Developer mode: Refreshing Docker image %s.", fullName)
		err := de.pullImage(fullName)
		if err != nil {
			log.Errorf("Developer mode: Refresh of Docker image %s failed: %s.", fullName, err)
			return err
		}
		image, _, err := de.client.ImageInspectWithRaw(context.Background(), fullName, false)
		if err != nil {
			log.Errorf("Developer mode: Docker image %s downloaded but can't be found locally: %s.", fullName, err)
			return err
		}
		log.Warnf("Developer mode: Docker image %s refreshed. Image id is %s.", fullName, shortImageID(image.ID))
	}
	return nil
}

func (de *DockerEngine) newEnvironment(bundle *config.Bundle) (circuit.Environment, error) {
	// Only happens if Relay is running in developer mode
	err := de.developerModeRefresh(bundle)
	if err != nil {
		return nil, err
	}
	client, err := newClient(de.config)
	if err != nil {
		return nil, err
	}
	options := circuit.CreateEnvironmentOptions{}
	options.Kind = circuit.DockerKind
	options.Bundle = bundle.Name
	options.DockerOptions.Conn = client
	options.DockerOptions.Image = bundle.Docker.Image
	options.DockerOptions.Tag = bundle.Docker.Tag
	options.DockerOptions.Binds = bundle.Docker.Binds
	options.DockerOptions.DriverInstance = "cog-circuit-driver"
	options.DockerOptions.DriverPath = "/operable/circuit/bin/circuit-driver"
	options.DockerOptions.Memory = int64(de.relayConfig.Docker.ContainerMemory * megabyte)
	return circuit.CreateEnvironment(options)
}

func (de *DockerEngine) needsUpdate(name, meta string) bool {
	fullName := fmt.Sprintf("%s:%s", name, meta)
	if meta != "latest" {
		image, _, _ := de.client.ImageInspectWithRaw(context.Background(), fullName, false)
		if image.ID != "" {
			log.Debugf("Resolved Docker image name %s to %s.", fullName, shortImageID(image.ID))
			return false
		}
	}
	return true
}

func (de *DockerEngine) pullImage(fullName string) error {
	err := de.ensureConnected()
	if err != nil {
		return err
	}
	closer, pullErr := de.client.ImagePull(context.Background(), fullName,
		types.ImagePullOptions{
			All:          false,
			RegistryAuth: de.auth,
		})
	if closer != nil {
		ioutil.ReadAll(closer)
		closer.Close()
	}
	return pullErr
}

func (de *DockerEngine) removeOldImage(oldID, newID, fullName string) {
	// Previous version of image existed and has been replaced.
	// Delete the old one to keep disk usage under control.
	if oldID != "" && oldID != newID {
		_, removeErr := de.client.ImageRemove(context.Background(), oldID,
			types.ImageRemoveOptions{
				Force:         true,
				PruneChildren: true,
			})
		if removeErr != nil {
			log.Errorf("Failed to remove old Docker image %s: %s.", shortImageID(oldID), removeErr)
		} else {
			log.Infof("Replaced obsolete Docker image %s with %s.", shortImageID(oldID), shortImageID(newID))
		}
	} else {
		if oldID != "" {
			log.Infof("Docker image %s for %s is up to date.", shortImageID(oldID), fullName)
		}
	}
}

func (de *DockerEngine) ensureConnected() error {
	if de.client == nil {
		client, err := newClient(de.config)
		if err != nil {
			log.Errorf("Failed to connect to Docker daemon: %s.", err)
			return err
		}
		de.client = client
	}
	return nil
}

func newClient(dockerConfig config.DockerInfo) (*client.Client, error) {
	if dockerConfig.UseEnv {
		client, err := client.NewEnvClient()
		if err != nil {
			return nil, err
		}
		return client, nil
	}
	dockerAPIVersion := os.Getenv("DOCKER_API_VERSION")
	if dockerAPIVersion == "" {
		dockerAPIVersion = client.DefaultVersion
	}
	client, err := client.NewClient(dockerConfig.SocketPath, dockerAPIVersion, nil, nil)
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
	if len(chunks) == 1 {
		return chunks[0][:11]
	}
	return chunks[1][:11]
}

func makeKey(pipelineID string, bundle *config.Bundle) string {
	return fmt.Sprintf("%s/%s:%s", pipelineID, bundle.Name, bundle.Version)
}
