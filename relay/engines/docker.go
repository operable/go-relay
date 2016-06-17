package engines

import (
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
	client        *client.Client
	relayConfig   *config.Config
	config        config.DockerInfo
	registryToken string
	cache         *envCache
}

// NewDockerEngine makes a new DockerEngine instance
func NewDockerEngine(relayConfig *config.Config, cache *envCache) (Engine, error) {
	dockerConfig := *relayConfig.Docker
	client, err := newClient(dockerConfig)
	if err != nil {
		return nil, err
	}
	return &DockerEngine{
		client:      client,
		relayConfig: relayConfig,
		config:      dockerConfig,
		cache:       cache,
	}, nil
}

// Init is required by the engines.Engine interface
func (de *DockerEngine) Init() error {
	err := de.attemptAuth()
	if err != nil {
		return err
	}
	return de.createCircuitDriver()
}

// IsAvailable returns true/false if a Docker image is found
func (de *DockerEngine) IsAvailable(name string, meta string) (bool, error) {
	err := de.attemptAuth()
	if err != nil {
		return false, err
	}
	fullName := fmt.Sprintf("%s:%s", name, meta)
	log.Debugf("Retrieving %s:%s from upstream Docker registry.", name, meta)
	beforeID, _ := de.IDForName(name, meta)
	closer, pullErr := de.client.ImagePull(context.Background(), fullName,
		types.ImagePullOptions{
			All:          false,
			RegistryAuth: de.registryToken,
		})
	if closer != nil {
		ioutil.ReadAll(closer)
		closer.Close()
	}
	if pullErr != nil {
		log.Errorf("Error ocurred when pulling image %s: %s.", name, pullErr)
		image, _, inspectErr := de.client.ImageInspectWithRaw(context.Background(), fullName, false)
		if inspectErr != nil {
			log.Errorf("Unable to find Docker image %s locally or in remote registry.", name)
			return false, pullErr
		}
		log.Infof("Retrieving Docker image %s from remote registry failed. Falling back to local copy, if it exists.", name)
		return image.ID != "", nil
	}
	afterID, err := de.IDForName(name, meta)
	if err != nil {
		log.Errorf("Failed to resolve image name %s:%s to an id: %s.", name, meta, err)
		return false, err
	}
	if beforeID == "" {
		log.Infof("Retrieved Docker image %s for %s.", shortImageID(afterID), fullName)
		return true, nil
	}
	if beforeID != afterID {
		_, removeErr := de.client.ImageRemove(context.Background(), beforeID,
			types.ImageRemoveOptions{
				Force:         true,
				PruneChildren: true,
			})
		if removeErr != nil {
			log.Errorf("Failed to remove old Docker image %s: %s.", shortImageID(beforeID), removeErr)
		} else {
			log.Infof("Replaced obsolete Docker image %s with %s.", shortImageID(beforeID), shortImageID(afterID))
		}
	} else {
		log.Infof("Docker image %s for %s is up to date.", shortImageID(beforeID), fullName)
	}
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
	image, _, err := de.client.ImageInspectWithRaw(context.Background(), fmt.Sprintf("%s:%s", name, meta), false)
	if err != nil {
		return "", err
	}
	return image.ID, nil
}

// Clean removes exited containers
func (de *DockerEngine) Clean() int {
	count := 0
	for _, env := range de.cache.getOld() {
		if env.Shutdown() == nil {
			count++
		}
	}
	args := filters.NewArgs()
	args.Add("status", "exited")
	args.Add(relayCreatedLabel, "yes")
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
	if de.registryToken == "" {
		authConfig := de.makeAuthConfig()
		if authConfig == nil {
			return nil
		}
		resp, err := de.client.RegistryLogin(context.Background(), *authConfig)
		if err != nil {
			log.Errorf("Authenticating to Docker registry failed: %s.", err)
			return err
		}
		de.registryToken = resp.IdentityToken
	}
	return nil
}

func (de *DockerEngine) newEnvironment(bundle *config.Bundle) (circuit.Environment, error) {
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
	options.DockerOptions.DriverInstance = "cog-circuit-driver"
	options.DockerOptions.DriverPath = "/operable/circuit/bin/circuit-driver"
	options.DockerOptions.Memory = int64(de.relayConfig.Docker.ContainerMemory * megabyte)
	return circuit.CreateEnvironment(options)
}

func verifyCredentials(client *client.Client, dockerConfig *config.DockerInfo) error {
	if dockerConfig.RegistryUser == "" || dockerConfig.RegistryPassword == "" || dockerConfig.RegistryEmail == "" {
		log.Info("No Docker registry credentials found or credentials are incomplete. Skipping auth check.")
		return nil
	}
	log.Info("Verifying Docker registry credentials.")
	authConf := types.AuthConfig{
		ServerAddress: dockerConfig.RegistryHost,
		Username:      dockerConfig.RegistryUser,
		Password:      dockerConfig.RegistryPassword,
		Email:         dockerConfig.RegistryEmail,
	}
	_, err := client.RegistryLogin(context.Background(), authConf)
	return err
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
	return chunks[1][:11]
}

func makeKey(pipelineID string, bundle *config.Bundle) string {
	return fmt.Sprintf("%s/%s:%s", pipelineID, bundle.Name, bundle.Version)
}
