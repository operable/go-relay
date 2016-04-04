package worker

import (
	"github.com/operable/go-relay/relay"
	"sync"
)

// RunWorker is a the logic loop for a request worker
func RunWorker(workQueue *relay.Queue, coordinator sync.WaitGroup) {
	coordinator.Add(1)
	defer coordinator.Done()
	for thing := workQueue.Dequeue(); thing != nil; thing = workQueue.Dequeue() {
	}
}
