package bundle

import (
	"github.com/coreos/go-semver/semver"
	"sort"
)

// LargestFirst sorts versions in descending order
type LargestFirst []*semver.Version

// Len is required by Sort.interface
func (nf LargestFirst) Len() int {
	return len(nf)
}

// Swap is required by Sort.interface
func (nf LargestFirst) Swap(i, j int) {
	nf[i], nf[j] = nf[j], nf[i]
}

// Less is required by Sort.interface
func (nf LargestFirst) Less(i, j int) bool {
	return nf[j].LessThan(*nf[i])
}

// VersionList is a list of semantic versions. It uses
// LargestFirst to keep versions in descending order.
type VersionList struct {
	members LargestFirst
}

// NewVersionList constructs an empty version list
func NewVersionList() *VersionList {
	vl := &VersionList{
		members: make(LargestFirst, 0),
	}
	return vl
}

// Add adds a new version to the list and ensures the list stays
// sorted in descending order.
func (vlp *VersionList) Add(version *semver.Version) {
	for _, v := range vlp.members {
		if *v == *version {
			return
		}
	}
	vlp.members = append(vlp.members, version)
	sort.Sort(vlp.members)
}

// Remove a version
func (vlp *VersionList) Remove(version *semver.Version) {
	candidate := version.String()
	for i, v := range vlp.members {
		if v.String() == candidate {
			vlp.members = append(vlp.members[:i], vlp.members[i+1:]...)
			break
		}
	}
}

// Len returns current size
func (vlp *VersionList) Len() int {
	return len(vlp.members)
}

// Largest returns the largest version contained in the list
func (vlp *VersionList) Largest() *semver.Version {
	if vlp.members.Len() == 0 {
		return nil
	}
	return vlp.members[0]
}
