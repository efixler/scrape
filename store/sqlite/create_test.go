package sqlite

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"testing"
)

func TestInMemoryDSN(t *testing.T) {
	_, err := dbPath(InMemoryDBName)
	if err != ErrIsInMemory {
		t.Errorf("expected ErrIsInMemory, got %v", err)
	}
}

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

func TestDbPath(t *testing.T) {
	type args struct {
		name        string
		filename    string
		expected    string
		expectedErr error
	}
	cwd, _ := os.Getwd()
	exec, _ := os.Executable()
	execPath, _ := filepath.Abs(filepath.Dir(exec))
	tests := []args{
		{"empty", "", filepath.Join(execPath, DefaultDatabase), nil},
		{"in memory", InMemoryDBName, InMemoryDBName, ErrIsInMemory},
		{"file no path", "foo.db", filepath.Join(cwd, "foo.db"), nil},
		{"file with relative path", "bar/foo.db", filepath.Join(cwd, "bar/foo.db"), nil},
		{"file with absolute path", "/baz/foo.db", "/baz/foo.db", nil},
	}
	for _, test := range tests {
		path, err := dbPath(test.filename)
		if path != test.expected {
			t.Errorf("%s: expected %s (for %s), got %s", test.name, test.expected, test.filename, path)
		}
		if err != test.expectedErr {
			t.Errorf("%s: expected error %v, got %v", test.name, test.expectedErr, err)
		}
	}
}

func TestAssertPathTo(t *testing.T) {
	type args struct {
		name        string
		path        string
		make        string
		expectedErr error
	}
	tests := []args{
		{"empty", "", "", nil},
		{"not empty", "foo", "", nil},
		{"unreachable", "bfile-xyz.txt/baz", "bfile-xyz.txt", ErrCantCreateDatabase},
	}
	deletes := make([]*os.File, 0)
	for _, test := range tests {
		if test.make != "" {
			file, err := os.Create(test.make)
			if err != nil {
				t.Fatalf("Error creating file %s: %v", test.path, err)
			}
			file.Close()
			deletes = append(deletes, file)
		}
		err := assertPathTo(test.path)
		if !errors.Is(err, test.expectedErr) {
			t.Errorf("%s: expected error %v, got %v", test.name, test.expectedErr, err)
		}
	}
	for _, file := range deletes {
		os.Remove(file.Name())
	}
}
