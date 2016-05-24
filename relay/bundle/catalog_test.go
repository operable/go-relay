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

var barBundle10 = config.Bundle{
	Name:    "bar",
	Version: "1.0.0",
}

func TestEmptyCatalog(t *testing.T) {
	bc := NewCatalog()
	if bc.Count() != 0 {
		t.Error("Expected empty catalog")
	}
	if bc.Find("foo") != nil {
		t.Error("Expected Find() to return nil")
	}
}

func TestCatalogWrite(t *testing.T) {
	bc := NewCatalog()
	if !bc.Replace([]*config.Bundle{&bundle12}) {
		t.Error("Expected Replace() to succeed")
	}
	if bc.Count() != 1 {
		t.Error("Expected Count() to return 1")
	}
	if !bc.IsChanged() {
		t.Error("Expected IsChanged() to return true")
	}
}

func TestCatalogFind(t *testing.T) {
	bc := NewCatalog()
	if !bc.Replace([]*config.Bundle{&bundle12}) {
		t.Error("Expected Replace() to succeed")
	}
	found := bc.Find(bundle12.Name)
	if found == nil || found.Name != bundle12.Name || found.Version != bundle12.Version {
		t.Error("Expected Find() to return stored bundle")
	}
}

func TestCatalogBatchAdds(t *testing.T) {
	bc := NewCatalog()
	batch := []*config.Bundle{&bundle12, &bundle121}
	if bc.Replace(batch) != true {
		t.Error("Batch update failed")
	}
	if bc.Len() != 1 {
		t.Errorf("Bad length: %d", bc.Len())
	}
	if bc.Replace([]*config.Bundle{&bundle121, &barBundle10}) != true {
		t.Error("Batch update failed")
	}
	if bc.Len() != 2 {
		t.Errorf("Bad length: %d", bc.Len())
	}
}

func TestCatalogRemove(t *testing.T) {
	bc := NewCatalog()
	bc.Replace([]*config.Bundle{&bundle12, &barBundle10})
	bc.Remove(bundle12.Name)
	if bc.Len() != 1 {
		t.Error("Bad length")
	}
	if bc.IsChanged() == false {
		t.Error("Change detection failed")
	}
}
