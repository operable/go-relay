package relay

import (
	"errors"
	"sync"
)

// A Queue is a Go channel with some basic in/out accounting
type Queue struct {
	queue    chan interface{}
	enqueued int64
	dequeued int64
	guard    sync.Mutex
	stopped  bool
}

type queueSignal byte

// NewQueue creates a new work queue
func NewQueue(size int) *Queue {
	return &Queue{
		queue:   make(chan interface{}, size),
		stopped: false,
	}
}

// Enqueue adds a work item to the queue.
// Returns an error if the queue is stopped.
func (q *Queue) Enqueue(thing interface{}) error {
	if q.stopped {
		return errors.New("Work queue is stopped")
	}
	q.queue <- thing
	q.updateEnqueued()
	return nil
}

// Dequeue removes the next item from the queue. Blocks
// when the queue is empty.
// Returns nil if the queue is stopped and empty.
func (q *Queue) Dequeue() interface{} {
	if isStopped, backlog := q.Status(); isStopped && backlog == 0 {
		return nil
	}
	thing := <-q.queue
	switch thing.(type) {
	case queueSignal:
		return nil
	default:
		q.updateDequeued()
		return thing
	}
}

// Status returns stopped flag and number of pending work items
func (q *Queue) Status() (bool, int64) {
	q.guard.Lock()
	defer q.guard.Unlock()
	backlog := q.enqueued - q.dequeued
	return q.stopped, backlog
}

// Stop prevents new work from being queued and allows
// consumers to drain remaining work
func (q *Queue) Stop() {
	if q.stopped == false {
		q.stopped = true
		q.queue <- queueSignal(0)
	}
}

func (q *Queue) updateDequeued() {
	q.guard.Lock()
	defer q.guard.Unlock()
	q.dequeued++
}

func (q *Queue) updateEnqueued() {
	q.guard.Lock()
	defer q.guard.Unlock()
	q.enqueued++
}
