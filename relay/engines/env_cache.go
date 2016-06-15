package engines

import (
	log "github.com/Sirupsen/logrus"
	"github.com/operable/circuit"
	"sync"
	"time"
)

var oldAge = time.Duration(10) * time.Second

type cacheEntry struct {
	env      circuit.Environment
	inUse    bool
	lastUsed time.Time
}

type envCache struct {
	envs map[string]*cacheEntry
	lock sync.Mutex
}

// NewEnvCache returns a newly constructed environment cache
func newEnvCache() *envCache {
	ec := &envCache{
		envs: make(map[string]*cacheEntry),
	}
	return ec
}

// Get returns the cached environment if it exists. Otherwise it returns
// nil
func (ec *envCache) get(key string) circuit.Environment {
	ec.lock.Lock()
	defer ec.lock.Unlock()
	retval := ec.envs[key]
	if retval != nil && retval.inUse == false {
		retval.inUse = true
		retval.lastUsed = time.Now()
		log.Debugf("Reusing environment for %s", key)
		return retval.env
	}
	return nil
}

// Put stores an environment with the specified key. Returns false if an
// environment has already been stored with the given key.
func (ec *envCache) put(key string, env circuit.Environment) bool {
	ec.lock.Lock()
	defer ec.lock.Unlock()
	entry := ec.envs[key]
	if entry == nil {
		entry := &cacheEntry{
			env:      env,
			inUse:    false,
			lastUsed: time.Now(),
		}
		ec.envs[key] = entry
		return true
	}
	if entry.env != env {
		return false
	}
	entry.inUse = false
	ec.envs[key] = entry
	return true
}

func (ec *envCache) getOld() []circuit.Environment {
	retval := []circuit.Environment{}
	ec.lock.Lock()
	defer ec.lock.Unlock()
	now := time.Now()
	for key, value := range ec.envs {
		if value.inUse == false {
			if now.Sub(value.lastUsed) > oldAge {
				delete(ec.envs, key)
				retval = append(retval, value.env)
			}
		}
	}
	return retval
}
