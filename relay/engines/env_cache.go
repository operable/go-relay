package engines

import (
	"github.com/operable/go-relay/relay/engines/exec"
	"sync"
	"time"
)

var cacheScanInterval = time.Duration(30) * time.Second
var oldAge = time.Duration(10) * time.Second

type cacheEntry struct {
	env      exec.Environment
	inUse    bool
	lastUsed time.Time
}

type envCache struct {
	envs    map[string]*cacheEntry
	lock    sync.Mutex
	cleaner *time.Timer
}

// NewEnvCache returns a newly constructed environment cache
func newEnvCache() *envCache {
	ec := &envCache{
		envs: make(map[string]*cacheEntry),
	}
	ec.cleaner = time.AfterFunc(cacheScanInterval, ec.expireOld)
	return ec
}

// Get returns the cached environment if it exists. Otherwise it returns
// nil
func (ec *envCache) get(key string) exec.Environment {
	ec.lock.Lock()
	defer ec.lock.Unlock()
	retval := ec.envs[key]
	if retval != nil && retval.inUse == false {
		retval.inUse = true
		retval.lastUsed = time.Now()
		return retval.env
	}
	return nil
}

// Put stores an environment with the specified key. Returns false if an
// environment has already been stored with the given key.
func (ec *envCache) put(key string, env exec.Environment) bool {
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

func (ec *envCache) expireOld() {
	ec.lock.Lock()
	defer ec.lock.Unlock()
	now := time.Now()
	for key, value := range ec.envs {
		if value.inUse == false {
			if now.Sub(value.lastUsed) > oldAge {
				value.env.Terminate(false)
				delete(ec.envs, key)
			}
		}
	}
	ec.cleaner = time.AfterFunc(cacheScanInterval, ec.expireOld)
}
