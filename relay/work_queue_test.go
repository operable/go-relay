package relay

import (
	"math/rand"
	"testing"
	"time"
)

func TestQueueDequeue(t *testing.T) {
	workQueue := NewWorkQueue(2)
	enqueued := time.Now()
	err := workQueue.Enqueue(enqueued)
	if err != nil {
		t.Fatal(err)
	}
	dequeued := workQueue.Dequeue()
	if enqueued != dequeued {
		t.Errorf("Work queue changed enqueued object: %v", dequeued)
	}
}

func TestMultiQueueDequeue(t *testing.T) {
	coord := make(chan interface{})
	workQueue := NewWorkQueue(2)
	go func() {
		thing := workQueue.Dequeue()
		coord <- thing
	}()
	time.Sleep(time.Duration(rand.Int31n(100)) * time.Millisecond)
	enqueued := time.Now()
	err := workQueue.Enqueue(enqueued)
	if err != nil {
		t.Fatal(err)
	}
	dequeued := <-coord
	if enqueued != dequeued {
		t.Errorf("Concurrent queue usage is broken. Enqueued %v and dequeued %v", enqueued, dequeued)
	}
}

func TestStoppedQueue(t *testing.T) {
	coord := make(chan interface{})
	workQueue := NewWorkQueue(2)
	go func() {
		thing := workQueue.Dequeue()
		coord <- thing
		thing = workQueue.Dequeue()
		coord <- thing
	}()
	enqueued := time.Now()
	workQueue.Enqueue(enqueued)
	workQueue.Stop()
	dequeued := <-coord
	if enqueued != dequeued {
		t.Errorf("Concurrent queue usage is broken. Enqueued %v and dequeued %v", enqueued, dequeued)
	}
	dequeued = <-coord
	if dequeued != nil {
		t.Errorf("Stopped empty queue dequeued an object: %v", dequeued)
	}
	if workQueue.Enqueue(time.Now()) == nil {
		t.Errorf("Enqueuing on a stopped queue didn't raise an error")
	}
}
