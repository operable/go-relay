package relay

import (
	"fmt"
	"github.com/coreos/go-semver/semver"
	"github.com/operable/go-relay/relay/config"
	"sync"
)

// BundleCatalog tracks installed and available bundles. Catalog
// reads and writes are guarded by a single RWMutex.
type BundleCatalog struct {
	lock          sync.RWMutex
	lastAnnounced uint64
	epoch         uint64
	versions      map[string]*VersionList
	bundles       map[string]*config.Bundle
}

// NewBundleCatalog returns a initialized and empty catalog.
func NewBundleCatalog() *BundleCatalog {
	bc := BundleCatalog{
		lastAnnounced: 0,
		epoch:         0,
		versions:      make(map[string]*VersionList),
		bundles:       make(map[string]*config.Bundle),
	}
	return &bc
}

// Add stores a config.Bundle instance in the catalog. Duplicates
// are not allowed. config.Bundle identity is composed of name
// and version.
func (bc *BundleCatalog) Add(bundle *config.Bundle) bool {
	key := bc.makeBundleKey(bundle)
	version, err := semver.NewVersion(bundle.Version)
	if err != nil {
		return false
	}
	bc.lock.Lock()
	defer bc.lock.Unlock()
	if bc.bundles[key] == nil {
		bc.bundles[key] = bundle
		bc.epoch++
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

// Remove deletes the named config.Bundle instance from the catalog.
func (bc *BundleCatalog) Remove(name string, version string) {
	key := bc.makeKey(name, version)
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
func (bc *BundleCatalog) Find(name string, version string) *config.Bundle {
	key := bc.makeKey(name, version)
	bc.lock.RLock()
	defer bc.lock.RUnlock()
	return bc.bundles[key]
}

// FindLatest returns the config.Bundle instance with the highest known semantic version.
// nil is returned if the entry doesn't exist.
func (bc *BundleCatalog) FindLatest(name string) *config.Bundle {
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
	key := bc.makeKey(name, latest.String())
	return bc.bundles[key]
}

// Count returns the number of stored entries.
func (bc *BundleCatalog) Count() int {
	bc.lock.RLock()
	defer bc.lock.RUnlock()
	return len(bc.bundles)
}

// ShouldAnnounce returns true if the catalog has been modified since
// the last announcement.
func (bc *BundleCatalog) ShouldAnnounce() bool {
	bc.lock.RLock()
	defer bc.lock.RUnlock()
	return bc.lastAnnounced != bc.epoch
}

func (bc *BundleCatalog) makeBundleKey(bundle *config.Bundle) string {
	return bc.makeKey(bundle.Name, bundle.Version)
}

func (bc *BundleCatalog) makeKey(name string, version string) string {
	return fmt.Sprintf("%s@%s", name, version)
}
