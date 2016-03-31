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

func NewDockerEngine(config *DockerInfo, wg *sync.WaitGroup) (Subsystem, error) {
	if config == nil {
		log.Info("Docker engine disabled")
		return &DisabledSubsystem{}, nil
	}
	if wg == nil {
		err := errors.New("Subsystem coordinator is nil")
		log.Fatal(err)
		return nil, err
	}
	client, err := connectToDocker(config)
	if err != nil {
		return nil, err
	}
	log.Info("Docker execution engine connected to Docker daemon")
	return &DockerEngine{
		config:  config,
		client:  client,
		wg:      wg,
		control: make(chan int, 1),
	}, nil
}

func (de *DockerEngine) Halt() error {
	de.control <- 1
	return nil
}

func (de *DockerEngine) Run() error {
	de.wg.Add(1)
	go func() {
		defer de.wg.Done()
		<-de.control
		log.Info("Docker execution engine halted")
	}()
	return nil
}

func (de *DockerEngine) Call(data interface{}) (interface{}, error) {
	return nil, errors.New("Not implemented")
}

func connectToDocker(config *DockerInfo) (*docker.Client, error) {
	if config.UseEnv {
		return docker.NewClientFromEnv()
	} else {
		return docker.NewClient(config.SocketPath)
	}
}
