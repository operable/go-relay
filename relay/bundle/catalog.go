package bundle

import (
	"fmt"
	log "github.com/Sirupsen/logrus"
	"github.com/coreos/go-semver/semver"
	"github.com/operable/go-relay/relay/config"
	"sync"
)

// Catalog tracks installed and available bundles. Catalog
// reads and writes are guarded by a single RWMutex.
type Catalog struct {
	lock      sync.RWMutex
	lastAcked uint64
	epoch     uint64
	versions  map[string]*VersionList
	bundles   map[string]*config.Bundle
}

// NewCatalog returns a initialized and empty catalog.
func NewCatalog() *Catalog {
	bc := Catalog{
		lastAcked: 0,
		epoch:     0,
		versions:  make(map[string]*VersionList),
		bundles:   make(map[string]*config.Bundle),
	}
	return &bc
}

// AddBatch adds new bundles to the catalog. Batch is
// processed atomically. Returns true if at least one config.Bundle
// entry was added.
func (bc *Catalog) AddBatch(bundles []*config.Bundle) bool {
	bc.lock.Lock()
	defer bc.lock.Unlock()
	dirty := false
	for _, bundle := range bundles {
		if bc.addToCatalog(bundle) == true && dirty == false {
			dirty = true
		}
	}
	if dirty == true {
		bc.epoch++
	}
	return dirty
}

// Add stores a config.Bundle instance in the catalog. Duplicates
// are not allowed. config.Bundle identity is composed of name
// and version.
func (bc *Catalog) Add(bundle *config.Bundle) bool {
	bc.lock.Lock()
	defer bc.lock.Unlock()
	if bc.addToCatalog(bundle) == true {
		bc.epoch++
		return true
	}
	return false
}

// Len returns the number of bundle versions stored
func (bc *Catalog) Len() int {
	bc.lock.RLock()
	defer bc.lock.RUnlock()
	return len(bc.bundles)
}

// BundleNames returns a unique list of bundles stored
func (bc *Catalog) BundleNames() []string {
	retval := make([]string, len(bc.versions))
	i := 0
	for k, _ := range bc.versions {
		retval[i] = k
		i++
	}
	return retval
}

// Remove deletes the named config.Bundle instance from the catalog.
func (bc *Catalog) Remove(name string, version string) {
	key := makeKey(name, version)
	sv, err := semver.NewVersion(version)
	if err != nil {
		return
	}
	bc.lock.Lock()
	defer bc.lock.Unlock()
	if bc.bundles[key] != nil {
		delete(bc.bundles, key)
		bc.epoch++
		bc.versions[name].Remove(sv)
	}
}

// Find retrieves a config.Bundle instance by name and version. nil is
// returned if the entry doesn't exist.
func (bc *Catalog) Find(name string, version string) *config.Bundle {
	key := makeKey(name, version)
	bc.lock.RLock()
	defer bc.lock.RUnlock()
	return bc.bundles[key]
}

// FindLatest returns the config.Bundle instance with the highest known semantic version.
// nil is returned if the entry doesn't exist.
func (bc *Catalog) FindLatest(name string) *config.Bundle {
	bc.lock.RLock()
	defer bc.lock.RUnlock()
	versions := bc.versions[name]
	if versions == nil {
		return nil
	}
	latest := versions.Largest()
	if latest == nil {
		return nil
	}
	key := makeKey(name, latest.String())
	return bc.bundles[key]
}

// Count returns the number of stored entries.
func (bc *Catalog) Count() int {
	bc.lock.RLock()
	defer bc.lock.RUnlock()
	return len(bc.bundles)
}

// ShouldAnnounce returns true if the catalog has been modified since
// the last announcement.
func (bc *Catalog) ShouldAnnounce() bool {
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

func (bc *Catalog) addToCatalog(bundle *config.Bundle) bool {
	key := makeKey(bundle.Name, bundle.Version)
	version, err := semver.NewVersion(bundle.Version)
	if err != nil {
		return false
	}
	if bc.bundles[key] == nil {
		bc.bundles[key] = bundle
		if bc.versions[bundle.Name] == nil {
			bc.versions[bundle.Name] = NewVersionList()
		}
		versions := bc.versions[bundle.Name]
		versions.Add(version)
		bc.versions[bundle.Name] = versions
		return true
	}
	return false
}

func (bc *Catalog) makeBundleKey(bundle *config.Bundle) string {
	return makeKey(bundle.Name, bundle.Version)
}

func makeKey(name string, version string) string {
	return fmt.Sprintf("%s@%s", name, version)
}
