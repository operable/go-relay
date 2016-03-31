package relay

import (
	"testing"
	"time"
)

type TestThing string

func TestSinglePoolTake(t *testing.T) {
	pool := newPool(2, t)
	if pool == nil {
		return
	}
	first := pool.Take(false)
	second := pool.Take(false)
	if first == second {
		t.Errorf("Take returns same object twice")
	}
	if third := pool.Take(false); third != nil {
		t.Errorf("Empty pool returns non-nil object")
	}
}

func TestSinglePoolTakeReturn(t *testing.T) {
	if pool := newPool(2, t); pool != nil {
		first := pool.Take(false)
		if first == nil {
			t.Errorf("Full pool returned nil")
		}
		pool.Return(first)
		first = pool.Take(false)
		if second := pool.Take(false); first != second {
			if third := pool.Take(false); second != third {
				if third != nil {
					t.Errorf("Empty pool returned object")
				}
			} else {
				t.Errorf("Pool returned duplicate object")
			}
		} else {
			t.Errorf("Pool returned duplicate object")
		}
	}
}

func TestEjectBadThing(t *testing.T) {
	if pool := newPool(2, t); pool != nil {
		first := pool.Take(false)
		if first == nil {
			t.Errorf("Full pool returned nil")
		}
		second := pool.Take(false)
		if second == nil {
			t.Errorf("Full pool returned nil")
		}
		pool.Eject(second)
		third := pool.Take(false)
		if third == second {
			t.Errorf("Eject failed")
		}
		if third == nil {
			t.Errorf("Pool didn't refill after ejection")
		}
	}
}

func newPool(size int, t *testing.T) *StaticResourcePool {
	pool, err := NewStaticPool(size, testMaker)
	if err != nil {
		t.Fatal(err)
		return nil
	}
	return pool
}

func testMaker() (interface{}, error) {
	return time.Now(), nil
}
