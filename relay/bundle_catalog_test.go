package relay

import (
	"github.com/coreos/go-semver/semver"
	"github.com/operable/go-relay/relay/config"
	"testing"
)

var bundle = config.Bundle{
	Name:    "foo",
	Version: "1.2.0",
}

var newerBundle = config.Bundle{
	Name:    "foo",
	Version: "1.2.1",
}

func TestEmptyBundleCatalog(t *testing.T) {
	bc := NewBundleCatalog()
	if bc.Count() != 0 {
		t.Error("Expected empty catalog")
	}
	if bc.Find("foo", "1.2") != nil {
		t.Error("Expected Find() to return nil")
	}
}

func TestBundleCatalogWrite(t *testing.T) {
	bc := NewBundleCatalog()
	if !bc.Add(&bundle) {
		t.Error("Expected Add() to succeed")
	}
	if bc.Count() != 1 {
		t.Error("Expected Count() to return 1")
	}
	if !bc.ShouldAnnounce() {
		t.Error("Expected ShouldAnnounce() to return true")
	}
}

func TestBundleCatalogFind(t *testing.T) {
	bc := NewBundleCatalog()
	if !bc.Add(&bundle) {
		t.Error("Expected Add() to succeed")
	}
	found := bc.Find(bundle.Name, bundle.Version)
	if found == nil || found.Name != bundle.Name || found.Version != bundle.Version {
		t.Error("Expected Find() to return stored bundle")
	}
}

func TestBundleCatalogFindLatest(t *testing.T) {
	bc := NewBundleCatalog()
	if !bc.Add(&newerBundle) {
		t.Error("Expected Add() to succeed")
	}
	latest := bc.FindLatest(newerBundle.Name)
	if latest == nil || latest.Name != newerBundle.Name || latest.Version != newerBundle.Version {
		t.Error("Expected FindLatest() to return newest bundle")
	}
	if !bc.Add(&bundle) {
		t.Error("Expected Add() to succeed")
	}
	latest2 := bc.FindLatest(bundle.Name)
	if latest != latest2 {
		t.Error("Expected FindLatest() to return newest bundle")
	}
}

func TestBundleCatalogFindLatest2(t *testing.T) {
	bc := NewBundleCatalog()
	if !bc.Add(&bundle) {
		t.Error("Expected Add() to succeed")
	}
	latest := bc.FindLatest(bundle.Name)
	if latest == nil || latest.Name != bundle.Name || latest.Version != bundle.Version {
		t.Error("Expected FindLatest() to return newest bundle")
	}
	if !bc.Add(&newerBundle) {
		t.Error("Expected Add() to succeed")
	}
	latest2 := bc.FindLatest(bundle.Name)
	if latest == latest2 {
		t.Error("Expected FindLatest() to return newest bundle")
	}
}

func TestAddVersionList(t *testing.T) {
	vl := NewVersionList()
	v1, _ := semver.NewVersion("1.0.0")
	v2, _ := semver.NewVersion("2.0.0")
	vl.Add(v1)
	if vl.Len() != 1 {
		t.Error("Bad length")
	}
	if vl.Largest() != v1 {
		t.Error("Bad Largest()")
	}
	vl.Add(v2)
	if vl.Len() != 2 {
		t.Error("Bad length")
	}
	if vl.Largest() != v2 {
		t.Error("Bad Largest()")
	}
}

func TestRemoveVersionList(t *testing.T) {
	vl := NewVersionList()
	v1, _ := semver.NewVersion("1.0.0")
	v2, _ := semver.NewVersion("2.0.0")
	vl.Add(v2)
	if vl.Len() != 1 {
		t.Error("Bad length")
	}
	if vl.Largest() != v2 {
		t.Error("Bad Largest()")
	}
	vl.Add(v1)
	if vl.Len() != 2 {
		t.Error("Bad length")
	}
	if vl.Largest() != v2 {
		t.Error("Bad Largest()")
	}
	vl.Remove(v2)
	if vl.Len() != 1 {
		t.Error("Bad length")
	}
	if vl.Largest() != v1 {
		t.Error("Bad Largest()")
	}
}
