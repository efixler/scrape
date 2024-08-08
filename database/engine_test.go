package database

import (
	"testing"
	"testing/fstest"
)

func TestBaseEngineImplementsInterfaces(t *testing.T) {
	var e any = NewEngine(
		"fakedriver",
		BaseDataSource("fakedsn"),
		new(fstest.MapFS),
	)
	be, ok := e.(Engine)
	if !ok {
		t.Errorf("NewEngine() does not implement Engine")
	}
	if be.Driver() != "fakedriver" {
		t.Errorf("NewEngine() did not set driver")
	}
	if be.DSNSource().DSN() != "fakedsn" {
		t.Errorf("NewEngine() did not set DSN")
	}
	if be.MigrationFS() == nil {
		t.Errorf("NewEngine() did not set migrationFS")
	}
}
