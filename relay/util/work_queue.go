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
	Stop(evt QueueEvents) bool
	Start()
	IsStopped() bool
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
	if wq.pending == 0 && wq.events != nil {
		events := wq.events
		go func() {
			events <- 1
		}()
		wq.events = nil
	}
	wq.lock.Unlock()
	return item, nil
}

func (wq *workQueue) Stop(evt QueueEvents) bool {
	wq.lock.Lock()
	defer wq.lock.Unlock()
	if wq.stopped {
		return false
	}
	wq.stopped = true
	if evt != nil {
		if wq.pending == 0 {
			go func() {
				evt <- 1
			}()
		} else {
			wq.events = evt
		}
	}
	return true
}

func (wq *workQueue) Start() {
	wq.lock.Lock()
	defer wq.lock.Unlock()
	if !wq.stopped {
		return
	}
	wq.stopped = false
	wq.events = nil
}

func (wq *workQueue) IsStopped() bool {
	wq.lock.Lock()
	defer wq.lock.Unlock()
	return wq.stopped && wq.pending == 0
}
