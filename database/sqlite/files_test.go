package sqlite

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/efixler/scrape/database"
	"github.com/efixler/scrape/store"
)

func TestDbPath(t *testing.T) {
	type args struct {
		name        string
		filename    string
		expected    string
		expectedErr error
	}
	cwd, _ := os.Getwd()
	tests := []args{
		{"empty", "", filepath.Join(cwd, DefaultDatabase), nil},
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
		{"unreachable", "bfile-xyz.txt/baz", "bfile-xyz.txt", store.ErrCantCreateDatabase},
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

// TODO: This test relies on how we're hardcoding the migrations directory in,
// but also providing in the engine interface. We should be testing here with
// and without the migrations directory.
func TestCreate(t *testing.T) {
	fname := "_test_create.db"
	engine, err := New(File(fname))
	if err != nil {
		t.Errorf("Error creating database engine: %v", err)
	}
	store := database.New(engine)
	err = store.Open(context.Background())
	if err != nil {
		t.Errorf("Error opening (and creating) database: %v", err)
	}
	// TODO: Test schema here
	defer func() {
		os.Remove(fname)
		os.Remove(fname + "-wal")
		os.Remove(fname + "-shm")
	}()
	_, err = os.Stat(fname)
	if os.IsNotExist(err) {
		t.Errorf("Database file not created")
	}
}
