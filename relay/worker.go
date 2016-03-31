package relay

import (
	"sync"
)

func RunWorker(workQueue *WorkQueue, coordinator sync.WaitGroup) {
	coordinator.Add(1)
	defer coordinator.Done()
	for thing := workQueue.Dequeue(); thing != nil; thing = workQueue.Dequeue() {
	}
}
