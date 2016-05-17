package relay

import (
	"github.com/coreos/go-semver/semver"
	"testing"
)

var v1, _ = semver.NewVersion("1.0.0")
var v11, _ = semver.NewVersion("1.1.0")
var v2, _ = semver.NewVersion("2.0.0")

func TestCreateEmptyVersionList(t *testing.T) {
	vl := NewVersionList()
	if vl.Len() != 0 {
		t.Error("Expected empty list to have 0 length")
	}
}

func TestAddVersion(t *testing.T) {
	vl := NewVersionList()
	vl.Add(v11)
	if vl.Len() != 1 {
		t.Error("Bad length")
	}
	if vl.Largest() != v11 {
		t.Error("Wrong version")
	}
}

func TestAddMultipleVersions(t *testing.T) {
	vl := NewVersionList()
	vl.Add(v11)
	if vl.Largest() != v11 {
		t.Error("Wrong version")
	}
	vl.Add(v1)
	if vl.Largest() != v11 {
		t.Error("Wrong version")
	}
}

func TestAddNewerVersion(t *testing.T) {
	vl := NewVersionList()
	vl.Add(v1)
	if vl.Largest() != v1 {
		t.Error("Wrong version")
	}
	vl.Add(v2)
	if vl.Largest() != v2 {
		t.Error("Wrong version")
	}
}

func TestRemoveVersion(t *testing.T) {
	vl := NewVersionList()
	vl.Add(v11)
	vl.Add(v2)
	vl.Add(v1)
	vl.Remove(v2)
	if vl.Len() != 2 {
		t.Error("Bad length")
	}
	if vl.Largest() != v11 {
		t.Error("Wrong version")
	}
}

func TestAddDuplicate(t *testing.T) {
	vl := NewVersionList()
	v1copy, _ := semver.NewVersion(v1.String())
	vl.Add(v1)
	vl.Add(v1)
	// Add a copy to test that VersionList isn't using pointer equality
	// to weed out duplicates
	vl.Add(v1copy)
	vl.Add(v11)
	if vl.Len() != 2 {
		t.Error("Bad length")
	}
}

func TestRemoveMissingVersion(t *testing.T) {
	vl := NewVersionList()
	vl.Add(v1)
	vl.Add(v11)
	vl.Add(v2)
	v11copy, _ := semver.NewVersion(v11.String())
	vl.Remove(v11copy)
	if vl.Len() != 2 {
		t.Error("Bad length")
	}
	vl.Remove(v11)
	if vl.Len() != 2 {
		t.Error("Bad length")
	}
}
