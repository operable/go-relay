package worker

import (
	"github.com/operable/go-relay/relay"
	"github.com/operable/go-relay/relay/bus"
	"github.com/operable/go-relay/relay/engines"
	"sync"
)

// Request represents an incoming message
// Can be either a Relay directive or a command execution
type Request struct {
	Bus     bus.MessageBus
	Engine  engines.Engine
	Topic   string
	Message []byte
}

func RunWorker(workQueue *relay.Queue, coordinator sync.WaitGroup) {
	coordinator.Add(1)
	defer coordinator.Done()
	for thing := workQueue.Dequeue(); thing != nil; thing = workQueue.Dequeue() {
	}
}
