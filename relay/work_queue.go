package relay

import (
	"errors"
	"sync"
)

// A WorkQueue is a Go channel with some basic in/out accounting
type WorkQueue struct {
	queue    chan interface{}
	enqueued int64
	dequeued int64
	guard    sync.Mutex
	stopped  bool
}

type queueSignal byte

// NewWorkQueue creates a new work queue
func NewWorkQueue(size int) *WorkQueue {
	return &WorkQueue{
		queue:   make(chan interface{}, size),
		stopped: false,
	}
}

// Enqueue adds a work item to the queue.
// Returns an error if the queue is stopped.
func (wq *WorkQueue) Enqueue(thing interface{}) error {
	if wq.stopped {
		return errors.New("WorkQueue is stopped")
	}
	wq.queue <- thing
	wq.enqueued++
	return nil
}

// Dequeue removes the next item from the queue. Blocks
// when the queue is empty.
// Returns nil if the queue is stopped and empty.
func (wq *WorkQueue) Dequeue() interface{} {
	if wq.stopped && wq.Backlog() == 0 {
		return nil
	}
	thing := <-wq.queue
	switch thing.(type) {
	case queueSignal:
		return nil
	default:
		wq.updateDequeued()
		return thing
	}
}

// Backlog returns the number of pending work items
func (wq *WorkQueue) Backlog() int64 {
	wq.guard.Lock()
	defer wq.guard.Unlock()
	return wq.enqueued - wq.dequeued
}

// Stop prevents new work from being queued and allows
// consumers to drain remaining work
func (wq *WorkQueue) Stop() {
	if wq.stopped == false {
		wq.stopped = true
		wq.queue <- queueSignal(0)
	}
}

func (wq *WorkQueue) updateDequeued() {
	wq.guard.Lock()
	defer wq.guard.Unlock()
	wq.dequeued++
}
