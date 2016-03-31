package relay

import (
	"container/list"
	"fmt"
	"sync"
)

// ResourceMaker populates ResourcePools
type ResourceMaker func() (interface{}, error)

// A StaticResourcePool is a fixed size resource pool
type StaticResourcePool struct {
	size     int
	maker    ResourceMaker
	alloc    *list.List
	free     *list.List
	waitChan chan byte
	guard    sync.Mutex
}

// IllegalPoolSize error
type IllegalPoolSize int

func (ips IllegalPoolSize) Error() string {
	return fmt.Sprintf("Pool size must be at least 1: %d", ips)
}

// NewStaticPool creates a new static pool
func NewStaticPool(size int, maker ResourceMaker) (*StaticResourcePool, error) {
	if size < 1 {
		return nil, IllegalPoolSize(size)
	}
	pool := StaticResourcePool{
		size:     size,
		maker:    maker,
		alloc:    list.New(),
		free:     list.New(),
		waitChan: make(chan byte),
	}
	for i := 0; i < size; i++ {
		thing, err := pool.maker()
		if err != nil {
			return nil, err
		}
		pool.free.PushFront(thing)
	}
	return &pool, nil
}

// Take a free thing from the pool. Returns nil if pool is empty
func (srp *StaticResourcePool) Take(wait bool) interface{} {
	if wait {
		return srp.takeOrWait()
	}
	srp.guard.Lock()
	defer srp.guard.Unlock()
	if srp.free.Len() == 0 {
		return nil
	}
	return srp.allocateThing()
}

// Return a thing to the pool
func (srp *StaticResourcePool) Return(thing interface{}) {
	srp.guard.Lock()
	defer srp.guard.Unlock()
	// Discard thing if pool is at max size
	if srp.free.Len() == srp.size {
		return
	}
	srp.freeThing(thing)
}

// Eject a defective thing and create a replacement
func (srp *StaticResourcePool) Eject(thing interface{}) error {
	srp.guard.Lock()
	defer srp.guard.Unlock()
	if srp.free.Len() == srp.size {
		return nil
	}
	replacement, err := srp.maker()
	if err != nil {
		return err
	}
	srp.free.PushFront(replacement)
	return nil
}

func (srp *StaticResourcePool) allocateThing() interface{} {
	thing := srp.free.Remove(srp.free.Front())
	srp.alloc.PushFront(thing)
	return thing
}

func (srp *StaticResourcePool) freeThing(thing interface{}) {
	for e := srp.alloc.Front(); e != nil; e = e.Next() {
		// We found thing so update accounting
		if e.Value == thing {
			srp.alloc.Remove(e)
			srp.free.PushBack(thing)
			break
		}
	}
}

func (srp *StaticResourcePool) takeOrWait() interface{} {
	srp.guard.Lock()
	for srp.free.Len() == srp.size {
		srp.guard.Unlock()
		<-srp.waitChan
		srp.guard.Lock()
	}
	defer srp.guard.Unlock()
	return srp.allocateThing()
}
