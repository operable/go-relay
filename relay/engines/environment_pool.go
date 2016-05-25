package engines

import (
	"errors"
	"github.com/operable/go-relay/relay/config"
	"github.com/operable/go-relay/relay/engines/exec"
	"sync"
	"time"
)

const (
	// MaxEnvUses is the number of times an environment can be
	// used before it must be terminated
	MaxUses = 20
)

// ErrorPoolClosed is returned when Acquire() is called on
// a closed pool.
var ErrorPoolClosed = errors.New("Pool is closed.")

// EnvironmentMaker is a function to make Environment
// instances
type EnvironmentMaker func() (exec.Environment, error)

// EnvironmentRecord contains metadata used by EnvironmentPool
// to manage individual Environment instances.
type EnvironmentRecord struct {
	env      exec.Environment
	usage    int
	lastUsed time.Time
}

// EnvironmentPool manages a shared pool of Environments
type EnvironmentPool struct {
	bundle *config.Bundle
	maker  EnvironmentMaker
	busy   []EnvironmentRecord
	idle   []EnvironmentRecord
	min    int
	max    int
	closed bool
	lock   sync.Mutex
}

// EnvironmentPoolCreateOptions are caller-settable options
// controlling various aspects of an EnvironmentPool
type EnvironmentPoolCreateOptions struct {
	Min    int
	Max    int
	Maker  EnvironmentMaker
	Bundle *config.Bundle
}

func NewEnvironmentPool(options EnvironmentPoolCreateOptions) (*EnvironmentPool, error) {
	enginePool := &EnvironmentPool{
		bundle: options.Bundle,
		maker:  options.Maker,
		min:    options.Min,
		max:    options.Max,
		closed: false,
		busy:   []EnvironmentRecord{},
		idle:   []EnvironmentRecord{},
	}
	if err := enginePool.fill(enginePool.min); err != nil {
		return nil, err
	}
	return enginePool, nil
}

func (ep *EnvironmentPool) Acquire() (exec.Environment, error) {
	ep.lock.Lock()
	defer ep.lock.Unlock()
	// Pool closed?
	if ep.closed == true {
		return nil, ErrorPoolClosed
	}
	if len(ep.idle) == 0 {
		// Should we allocate more to the pool?
		totalSize := len(ep.idle) + len(ep.busy)
		if totalSize < ep.max {
			if err := ep.fill(fillSize(totalSize, ep.max)); err != nil {
				return nil, err
			}
		}
	}
	// If idle has capacity then allocate from there
	if len(ep.idle) > 0 {
		retval := ep.idle[0]
		retval.usage++
		retval.lastUsed = time.Now()
		ep.idle = ep.idle[1:]
		ep.busy = append(ep.busy, retval)
		return retval.env, nil
	}
	// Allocate burst capacity
	env, err := ep.maker()
	if err != nil {
		return nil, err
	}
	return env, nil
}

func (ep *EnvironmentPool) Release(env exec.Environment) {
	ep.lock.Lock()
	defer ep.lock.Unlock()
	// If burst allocated terminate the environment
	record := ep.findBusyRecord(env)
	if record == nil {
		env.Terminate(true)
		return
	}
	if record.usage == MaxUses || ep.closed {
		env.Terminate(true)
		return
	}
	ep.idle = append(ep.idle, *record)
}

func (ep *EnvironmentPool) Remove(env exec.Environment) {
	ep.lock.Lock()
	defer ep.lock.Unlock()
	ep.findBusyRecord(env)
	env.Terminate(true)
}

func (ep *EnvironmentPool) Close() {
	ep.lock.Lock()
	defer ep.lock.Unlock()
	ep.closed = true
	// Remove all idle instances
	for _, idle := range ep.idle {
		idle.env.Terminate(true)
	}
	ep.idle = []EnvironmentRecord{}
}

func (ep *EnvironmentPool) findBusyRecord(env exec.Environment) *EnvironmentRecord {
	pos := -1
	for i, v := range ep.busy {
		if v.env == env {
			pos = i
			break
		}
	}
	if pos == -1 {
		return nil
	}
	retval := ep.busy[pos]
	ep.busy = append(ep.busy[:pos], ep.busy[pos+1:]...)
	return &retval
}

func (ep *EnvironmentPool) fill(n int) error {
	failed := error(nil)
	for i := 0; i < n; i++ {
		env, err := ep.maker()
		if err != nil {
			failed = err
			break
		}
		ep.idle = append(ep.idle, EnvironmentRecord{
			env:      env,
			usage:    0,
			lastUsed: time.Now(),
		})
	}
	if failed != nil {
		for _, record := range ep.idle {
			record.env.Terminate(true)
		}
	}
	return failed
}

func fillSize(current, max int) int {
	idealSize := (max - current) / 4
	if idealSize < 1.0 {
		return 1
	}
	return int(idealSize)
}
