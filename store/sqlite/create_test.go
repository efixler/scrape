package sqlite

import (
	"context"
	"os"
	"testing"
)

func TestCreate(t *testing.T) {
	fname := "_test_create.db"
	store, err := Factory(fname)()
	if err != nil {
		t.Errorf("Error creating database factory: %v", err)
	}
	err = store.Open(context.Background())
	if err != nil {
		t.Errorf("Error opening (and creating) database: %v", err)
	}
	defer os.Remove(fname)
	_, err = os.Stat(fname)
	if os.IsNotExist(err) {
		t.Errorf("Database file not created")
	}
}

func TestDontCreateWhenExists(t *testing.T) {
	t.Skip("Skipping test because we can no longer easily catch this condition, since the db will now autocreate")
	fname := "_test_dont_overwrite.db"
	if _, err := os.Stat(fname); !os.IsNotExist(err) {
		t.Fatalf("Database file %s already exists, can't run this test", fname)
	}
	_, err := os.Create(fname)
	if err != nil {
		t.Fatalf("Error creating dummy file %s: %v", fname, err)
	}
	defer os.Remove(fname)
	store, err := Factory(fname)()
	if err != nil {
		t.Errorf("Error creating store: %v", err)
	}
	err = store.Open(context.TODO())
	if err == nil {
		t.Errorf("Oops! Overwrote existing database: %v", err)
	}
}
