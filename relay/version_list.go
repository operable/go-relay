package relay

import (
	"github.com/coreos/go-semver/semver"
	"sort"
)

// List of semantic versions sorted in descending order
type NewestFirst []*semver.Version

func (nf NewestFirst) Len() int {
	return len(nf)
}

func (nf NewestFirst) Swap(i, j int) {
	nf[i], nf[j] = nf[j], nf[i]
}

func (nf NewestFirst) Less(i, j int) bool {
	return nf[j].LessThan(*nf[i])
}

// VersionList is a list of semantic versions sorted in descending
// order.
type VersionList struct {
	members NewestFirst
}

// NewVersionList constructs an empty version list
func NewVersionList() *VersionList {
	vl := &VersionList{
		members: make(NewestFirst, 0),
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
