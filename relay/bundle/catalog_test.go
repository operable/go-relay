package bundle

import (
	"github.com/operable/go-relay/relay/config"
	"testing"
)

var bundle12 = config.Bundle{
	Name:    "foo",
	Version: "1.2.0",
}

var bundle121 = config.Bundle{
	Name:    "foo",
	Version: "1.2.1",
}

var bundle13 = config.Bundle{
	Name:    "foo",
	Version: "1.3.0",
}

func TestEmptyCatalog(t *testing.T) {
	bc := NewCatalog()
	if bc.Count() != 0 {
		t.Error("Expected empty catalog")
	}
	if bc.Find("foo", "1.2") != nil {
		t.Error("Expected Find() to return nil")
	}
}

func TestCatalogWrite(t *testing.T) {
	bc := NewCatalog()
	if !bc.Add(&bundle12) {
		t.Error("Expected Add() to succeed")
	}
	if bc.Count() != 1 {
		t.Error("Expected Count() to return 1")
	}
	if !bc.ShouldAnnounce() {
		t.Error("Expected ShouldAnnounce() to return true")
	}
}

func TestCatalogWriteDupe(t *testing.T) {
	bc := NewCatalog()
	if !bc.Add(&bundle12) {
		t.Error("Expected Add() to succeed")
	}
	if bc.Add(&bundle12) {
		t.Error("Expected Add() with duplicate to fail")
	}
	if bc.Len() != 1 {
		t.Error("Bad length")
	}
}

func TestCatalogFind(t *testing.T) {
	bc := NewCatalog()
	if !bc.Add(&bundle12) {
		t.Error("Expected Add() to succeed")
	}
	found := bc.Find(bundle12.Name, bundle12.Version)
	if found == nil || found.Name != bundle12.Name || found.Version != bundle12.Version {
		t.Error("Expected Find() to return stored bundle")
	}
}

func TestCatalogFindLatest(t *testing.T) {
	bc := NewCatalog()
	if !bc.Add(&bundle121) {
		t.Error("Expected Add() to succeed")
	}
	latest := bc.FindLatest(bundle121.Name)
	if latest == nil || latest.Name != bundle121.Name || latest.Version != bundle121.Version {
		t.Error("Expected FindLatest() to return newest bundle")
	}
	if !bc.Add(&bundle12) {
		t.Error("Expected Add() to succeed")
	}
	latest2 := bc.FindLatest(bundle12.Name)
	if latest != latest2 {
		t.Error("Expected FindLatest() to return newest bundle")
	}
}

func TestCatalogFindLatest2(t *testing.T) {
	bc := NewCatalog()
	if !bc.Add(&bundle12) {
		t.Error("Expected Add() to succeed")
	}
	latest := bc.FindLatest(bundle12.Name)
	if latest == nil || latest.Name != bundle12.Name || latest.Version != bundle12.Version {
		t.Error("Expected FindLatest() to return newest bundle")
	}
	if !bc.Add(&bundle121) {
		t.Error("Expected Add() to succeed")
	}
	latest2 := bc.FindLatest(bundle12.Name)
	if latest == latest2 {
		t.Error("Expected FindLatest() to return newest bundle")
	}
}

func TestCatalogBatchAdds(t *testing.T) {
	bc := NewCatalog()
	batch := []*config.Bundle{&bundle12, &bundle121}
	if bc.AddBatch(batch) != true {
		t.Error("Batch update failed")
	}
	if bc.Len() != 2 {
		t.Error("Bad length")
	}
}

func TestCatalogBatchAddSingle(t *testing.T) {
	bc := NewCatalog()
	bc.Add(&bundle12)
	if bc.Len() != 1 {
		t.Error("Bad length")
	}
	batch := []*config.Bundle{&bundle121, &bundle13}
	if bc.AddBatch(batch) != true {
		t.Error("Batch update failed")
	}
	if bc.Len() != 3 {
		t.Error("Bad length")
	}
	if bc.FindLatest(bundle12.Name) != &bundle13 {
		t.Error("Expected FindLatest() to return newest bundle")
	}
}

func TestCatalogBatchAddDupes(t *testing.T) {
	bc := NewCatalog()
	bc.Add(&bundle12)
	batch := []*config.Bundle{&bundle12, &bundle121}
	if bc.AddBatch(batch) != true {
		t.Error("Batch update failed")
	}
	if bc.Len() == 3 {
		t.Error("De-duplication failed")
	}
	found := bc.FindLatest(bundle12.Name)
	if found.Name != bundle121.Name || found.Version != bundle121.Version {
		t.Error("Expected FindLatest() to return newest bundle")
	}
}
