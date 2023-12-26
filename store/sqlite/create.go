package sqlite

import (
	_ "embed"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
)

const (
	DefaultDatabase = "scrape_data/scrape.db"
	InMemoryDBName  = ":memory:"
)

var (
	ErrDatabaseExists = errors.New("database already exists")
	ErrIsInMemory     = errors.New("file path is in-memory DB (':memory:')")
)

// dbPath returns the path to the database file. If filename is empty,
// the path to the executable + the default path is returned.
// If filename is not empty filename is returned and its
// existence is checked.
func dbPath(filename string) (string, error) {
	switch filename {
	case InMemoryDBName:
		return InMemoryDBName, ErrIsInMemory
	case "":
		root, err := os.Executable()
		if err != nil {
			return "", err
		}
		root, err = filepath.Abs(root)
		if err != nil {
			return "", err
		}
		filename = filepath.Join(root, DefaultDatabase)
	}
	return filepath.Abs(filename)
}

func exists(fqn string) bool {
	if _, err := os.Stat(fqn); errors.Is(err, fs.ErrNotExist) {
		return false
	}
	return true
}

func (s SqliteStore) createPathToDB() error {
	dir := filepath.Dir(s.resolvedPath)
	if dh, _ := os.Stat(dir); dh == nil {
		err := os.MkdirAll(dir, 0775)
		if err != nil {
			return err
		}
	} else if !dh.IsDir() {
		return fmt.Errorf("path %s exists but is not a directory", dir)
	}
	return nil
}

//go:embed create.sql
var createSQL string

// When this is called, the path to the database must already exist.
func (s *SqliteStore) create() error {
	_, err := s.DB.ExecContext(s.Ctx, createSQL)
	return err
}

// func (s *SqliteStore) clear() error {
// 	if s.DB == nil {
// 		return ErrStoreNotOpen
// 	}
// 	if _, err := s.DB.ExecContext(s.Ctx, qClear); err != nil {
// 		return err
// 	}
// 	return nil
// }
