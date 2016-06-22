package util

import (
	"errors"
	"sync"
)

var errorQueueStopped = errors.New("Queue is stopped")

// Queue is a asynchronous work queue
type Queue interface {
	Enqueue(interface{}) error
	Dequeue() (interface{}, error)
}

// QueueEvents is used to notify waiting processes about
// administrative events. This is currently only used to
// notify when the queue is drained.
type QueueEvents chan byte

type workQueue struct {
	lock    sync.Mutex
	depth   uint
	pending uint
	events  QueueEvents
	stopped bool
	queue   chan interface{}
}

// NewQueue constructs a new queue instance with the
// specified depth.
func NewQueue(depth uint) Queue {
	return &workQueue{
		depth:   depth,
		pending: 0,
		stopped: true,
		queue:   make(chan interface{}, depth+1),
	}
}

func (wq *workQueue) Enqueue(item interface{}) error {
	wq.queue <- item
	wq.lock.Lock()
	defer wq.lock.Unlock()
	wq.pending++
	return nil
}

func (wq *workQueue) Dequeue() (interface{}, error) {
	item := <-wq.queue
	wq.lock.Lock()
	wq.pending--
	wq.lock.Unlock()
	return item, nil
}

func (wq *workQueue) decrementPending() {
	wq.lock.Lock()
	defer wq.lock.Unlock()
	wq.pending--
}

func (wq *workQueue) incrementPending() {
	wq.lock.Lock()
	defer wq.lock.Unlock()
	wq.pending++
}
