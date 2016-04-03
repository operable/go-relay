package worker

import (
	"github.com/operable/go-relay/relay/docker"
	"sync"
)

// Service is any independently running subsystem
type Service interface {
	Run() error
	Halt()
}

// MessageBus is the interface used by worker code to
// publish messages
type MessageBus interface {
	Publish(topic string, payload []byte) error
}

// Request represents an incoming message
// Can be either a Relay directive or a command execution
type Request struct {
	Bus          MessageBus
	DockerEngine *docker.Engine
	Topic        string
	Message      []byte
}

func RunWorker(workQueue *Queue, coordinator sync.WaitGroup) {
	coordinator.Add(1)
	defer coordinator.Done()
	for thing := workQueue.Dequeue(); thing != nil; thing = workQueue.Dequeue() {
	}
}
