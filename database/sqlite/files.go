package sqlite

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"github.com/efixler/scrape/store"
)

const (
	InMemoryDBName = ":memory:"
)

var (
	DefaultDatabase   = "scrape_data/scrape.db"
	ErrDatabaseExists = errors.New("database already exists")
	ErrIsInMemory     = errors.New("file path is in-memory DB (':memory:')")
)

// dbPath returns the absolute path to the database file. If filename is empty,
// the path to the current working directory + the default db filename is returned.
// For the special case of ':memory:', which is an in-memory db, as ErrIsInMemory
// is returned.
func dbPath(filename string) (string, error) {
	switch filename {
	case InMemoryDBName:
		return InMemoryDBName, ErrIsInMemory
	case "":
		filename = DefaultDatabase
	}
	return filepath.Abs(filename)
}

func exists(fqn string) bool {
	if _, err := os.Stat(fqn); err != nil {
		return false
	}
	// TODO: Revisit this. fs.ErrNotExist is only returned when the
	// last element doesn't exist, bbut when an intermediate path is a file
	// it returns a different error.
	return true
}

func assertPathTo(fqn string) error {
	if exists(fqn) {
		return nil
	}
	dir := filepath.Dir(fqn)
	if dh, _ := os.Stat(dir); dh == nil {
		err := os.MkdirAll(dir, 0775)
		if err != nil {
			return err
		}
	} else if !dh.IsDir() {
		return errors.Join(
			store.ErrCantCreateDatabase,
			fmt.Errorf("path %s exists but is not a directory", dir),
		)
	}
	return nil
}
