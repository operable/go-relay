package bundle

import (
	log "github.com/Sirupsen/logrus"
	"github.com/operable/go-relay/relay/config"
	"sync"
)

// Catalog tracks installed and available bundles. Catalog
// reads and writes are guarded by a single RWMutex.
type Catalog struct {
	lock      sync.RWMutex
	lastAcked uint64
	epoch     uint64
	bundles   map[string]*config.Bundle
}

// NewCatalog returns a initialized and empty catalog.
func NewCatalog() *Catalog {
	bc := Catalog{
		lastAcked: 0,
		epoch:     0,
		bundles:   make(map[string]*config.Bundle),
	}
	return &bc
}

type diffResult struct {
	added   []*config.Bundle
	removed []string
}

func newDiffResult() *diffResult {
	return &diffResult{
		added:   []*config.Bundle{},
		removed: []string{},
	}
}

// Replace atomically replaces the current bundle catalog contents with
// new a new snapshot. Returns true if at least one config.Bundle
// entry was added or deleted.
func (bc *Catalog) Replace(bundles []*config.Bundle) bool {
	bc.lock.Lock()
	defer bc.lock.Unlock()
	dirty := false
	result := bc.diff(bundles)
	log.Debugf("Updating bundle catalog. Adds: %d, Deletions: %d.", len(result.added), len(result.removed))
	for _, name := range result.removed {
		before := len(bc.bundles)
		delete(bc.bundles, name)
		after := len(bc.bundles)
		if !dirty && before != after {
			dirty = true
		}
	}

	for _, newBundle := range result.added {
		bc.bundles[newBundle.Name] = newBundle
		if !dirty {
			dirty = true
		}
	}
	if dirty == true {
		bc.epoch++
	}
	return dirty
}

// Len returns the number of bundle versions stored
func (bc *Catalog) Len() int {
	bc.lock.RLock()
	defer bc.lock.RUnlock()
	return len(bc.bundles)
}

// BundleNames returns a unique list of bundles stored
func (bc *Catalog) BundleNames() []string {
	bc.lock.RLock()
	defer bc.lock.RUnlock()
	names := []string{}
	for k := range bc.bundles {
		names = append(names, k)
	}
	return names
}

// Remove deletes the named config.Bundle instance from the catalog.
func (bc *Catalog) Remove(name string) {
	bc.lock.Lock()
	defer bc.lock.Unlock()
	if bc.bundles[name] != nil {
		delete(bc.bundles, name)
		bc.epoch++
	}
}

// Find retrieves a config.Bundle instance by name.
// returned if the entry doesn't exist.
func (bc *Catalog) Find(name string) *config.Bundle {
	bc.lock.RLock()
	defer bc.lock.RUnlock()
	return bc.bundles[name]
}

// Count returns the number of stored entries.
func (bc *Catalog) Count() int {
	bc.lock.RLock()
	defer bc.lock.RUnlock()
	return len(bc.bundles)
}

// IsChanged returns true if the catalog has been modified since
// the last announcement.
func (bc *Catalog) IsChanged() bool {
	bc.lock.RLock()
	defer bc.lock.RUnlock()
	return bc.lastAcked < bc.epoch
}

// CurrentEpoch returns the catalog's current epoch
func (bc *Catalog) CurrentEpoch() uint64 {
	bc.lock.RLock()
	defer bc.lock.RUnlock()
	return bc.epoch
}

// EpochAcked updates the catalog's state to reflect the latest
// acked epoch
func (bc *Catalog) EpochAcked(acked uint64) {
	if acked > bc.epoch {
		log.Warnf("Ignored bundle catalog epoch ack from the future. Current bundle catalog epoch is %d; acked epoch is %d.",
			bc.epoch, acked)
		return
	}
	bc.lastAcked = acked
}

// MarkReady updates the catalog entry to indicate the bundle is
// ready for execution
func (bc *Catalog) MarkReady(name string) {
	bc.lock.Lock()
	defer bc.lock.Unlock()
	if bundle := bc.bundles[name]; bundle != nil {
		bundle.SetAvailable(true)
	}
}

// Reconnected increments the catalog's epoch to indicate the Relay
// has reconnected to Cog and should re-announce.
func (bc *Catalog) Reconnected() {
	bc.lock.Lock()
	defer bc.lock.Unlock()
	bc.epoch++
}

func (bc *Catalog) diff(bundles []*config.Bundle) *diffResult {
	result := newDiffResult()

	// Iterate over existing catalog to find any removed
	// or updated (new version != current version) bundles
	for name, bundle := range bc.bundles {
		b := findByName(name, bundles)
		if b == nil {
			result.removed = append(result.removed, name)
		} else {
			// Delete existing and add new if versions don't match
			if b.Version != bundle.Version {
				result.added = append(result.added, b)
				result.removed = append(result.removed, name)
			}
		}
	}

	// Iterate over update to find completely new bundles
	for _, b := range bundles {
		// Add if in update and not in catalog
		if _, ok := bc.bundles[b.Name]; ok == false {
			result.added = append(result.added, b)
		}
	}
	return result
}

func findByName(name string, bundles []*config.Bundle) *config.Bundle {
	for _, bundle := range bundles {
		if bundle.Name == name {
			return bundle
		}
	}
	return nil
}
