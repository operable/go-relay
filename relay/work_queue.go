package relay

import (
	"errors"
	"sync"
)

var errorQueueStopped = errors.New("Queue is stopped")

// WorkQueue is an interface used by Relay
// and workers.
type Queue interface {
	Enqueue(interface{}) error
	Dequeue() (interface{}, error)
	Stop(bool)
	Start()
	IsStopped() bool
}

type workQueue struct {
	lock    sync.Mutex
	depth   uint
	pending uint
	drain   bool
	stopped bool
	queue   chan interface{}
}

func NewQueue(depth uint) Queue {
	return &workQueue{
		depth:   depth,
		pending: 0,
		stopped: false,
		queue:   make(chan interface{}, depth+1),
	}
}

func (wq *workQueue) Enqueue(item interface{}) error {
	if wq.IsStopped() {
		return errorQueueStopped
	}
	wq.queue <- item
	wq.lock.Lock()
	wq.pending++
	wq.lock.Unlock()
	return nil
}

func (wq *workQueue) Dequeue() (interface{}, error) {
	if wq.IsStopped() {
		return nil, errorQueueStopped
	}
	item := <-wq.queue
	wq.lock.Lock()
	wq.pending--
	wq.lock.Unlock()
	return item, nil
}

func (wq *workQueue) Stop(drain bool) {
	wq.lock.Lock()
	defer wq.lock.Unlock()
	if !wq.stopped {
		wq.stopped = true
		wq.drain = drain
	}
}

func (wq *workQueue) Start() {
	wq.lock.Lock()
	defer wq.lock.Unlock()
	if !wq.stopped {
		return
	}
	wq.stopped = false
	wq.drain = false
}

func (wq *workQueue) IsStopped() bool {
	wq.lock.Lock()
	defer wq.lock.Unlock()
	if wq.stopped {
		if wq.drain == true {
			return wq.pending == 0
		}
		return true
	}
	return false
}
