package sqlite

import (
	"context"
	"os"
	"testing"
)

func TestCreate(t *testing.T) {
	fname := "_test_create.db"
	err := CreateDB(context.Background(), fname)
	if err != nil {
		t.Errorf("Error creating database: %v", err)
	}
	defer os.Remove(fname)
	_, err = os.Stat(fname)
	if os.IsNotExist(err) {
		t.Errorf("Database file not created")
	}
}

func TestDontCreateWhenExists(t *testing.T) {
	fname := "_test_dont_overwrite.db"
	if _, err := os.Stat(fname); !os.IsNotExist(err) {
		t.Fatalf("Database file %s already exists, can't run this test", fname)
	}
	_, err := os.Create(fname)
	if err != nil {
		t.Fatalf("Error creating dummy file %s: %v", fname, err)
	}
	defer os.Remove(fname)
	err = CreateDB(context.TODO(), fname)
	if err == nil {
		t.Errorf("Oops! Overwrote existing database: %v", err)
	}
}
