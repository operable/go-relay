package relay

import (
	"math/rand"
	"testing"
	"time"
)

func TestMultiplePoolTakeReturn(t *testing.T) {
	pool := newPool(2, t)
	if pool == nil {
		return
	}
	output := make(chan interface{})
	for i := 0; i < 2; i++ {
		go func() {
			worker(pool, output)
		}()
	}
	first := <-output
	second := <-output
	if first == second {
		t.Errorf("Pool returned duplicate objects to concurrent callers")
	}
	if first == nil || second == nil {
		t.Errorf("Pool returned nil when resources should be available")
	}
	third := pool.Take(false)
	if third != nil {
		t.Errorf("Empty pool returned object")
	}
}

func worker(pool *StaticResourcePool, output chan interface{}) {

	for i := 0; i < 50; i++ {
		time.Sleep(time.Duration(rand.Int31n(43)) * time.Millisecond)
		thing := pool.Take(false)
		time.Sleep(time.Duration(rand.Int31n(43)) * time.Millisecond)
		pool.Return(thing)
	}
	output <- pool.Take(false)
}
