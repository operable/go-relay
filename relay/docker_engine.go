package relay

import (
	"errors"
	"sync"

	log "github.com/Sirupsen/logrus"
	"github.com/fsouza/go-dockerclient"
)

type DockerEngine struct {
	config  *DockerInfo
	client  *docker.Client
	wg      *sync.WaitGroup
	control chan int
}

func NewDockerEngine(config *DockerInfo, wg *sync.WaitGroup) (*DockerEngine, error) {
	if config == nil || wg == nil {
		err := errors.New("Docker config or coordination wait group is nil")
		log.Fatal(err)
		return nil, err
	}
	client, err := connectToDocker(config)
	if err != nil {
		return nil, err
	}
	return &DockerEngine{
		config:  config,
		client:  client,
		wg:      wg,
		control: make(chan int, 1),
	}, nil
}

func (de *DockerEngine) Stop() {
	de.control <- 1
}

func (de *DockerEngine) Run() error {
	de.wg.Add(1)
	go func() {
		defer de.wg.Done()
		<-de.control
	}()
	return nil
}

func (de *DockerEngine) CountImages() (int, error) {
	images, err := de.client.ListImages(docker.ListImagesOptions{})
	if err != nil {
		return 0, err
	}
	return len(images), nil
}

func connectToDocker(config *DockerInfo) (*docker.Client, error) {
	if config.UseEnv {
		return docker.NewClientFromEnv()
	} else {
		return docker.NewClient(config.SocketPath)
	}
}
