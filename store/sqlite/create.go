package sqlite

import (
	"context"
	"database/sql"
	_ "embed"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"time"
)

const (
	InMemoryDBName = ":memory:"
)

var (
	DefaultDatabase   = "scrape_data/scrape.db"
	ErrDatabaseExists = errors.New("database already exists")
	ErrIsInMemory     = errors.New("file path is in-memory DB (':memory:')")
)

// dbPath returns the path to the database file. If filename is empty,
// the path to the executable + the default path is returned.
// If filename is not empty filename is returned and its
// existence is checked.
// TODO: Pull the ""/executable assumptions out of this function and into options.
func dbPath(filename string) (string, error) {
	switch filename {
	case InMemoryDBName:
		return InMemoryDBName, ErrIsInMemory
	case "":
		root, err := os.Executable()
		fmt.Printf("Root: %s\n", root)
		if err != nil {
			return "", err
		}
		root, err = filepath.Abs(filepath.Dir(root))
		fmt.Printf("Root filepath: %s\n", root)
		if err != nil {
			return "", err
		}
		filename = filepath.Join(root, DefaultDatabase)
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
			ErrCantCreateDatabase,
			fmt.Errorf("path %s exists but is not a directory", dir),
		)
	}
	return nil
}

//go:embed create.sql
var createSQL string

// When this is called, the path to the database must already exist.
func (s *SqliteStore) Create() error {
	_, err := s.DB.ExecContext(s.Ctx, createSQL)
	if err != nil {
		slog.Error("sqlite: error creating database", "error", err)
	}
	return err
}

// Private version of the maintenance function that doesn't log, for running
// on the timer provided by DBHandle.
func maintain(ctx context.Context, db *sql.DB, tm time.Time) error {
	slog.Debug("sqlite: maintenance ran", "time", tm)
	_, err := db.ExecContext(ctx, maintenanceSQL)
	return err
}

//go:embed maintenance.sql
var maintenanceSQL string

func (s *SqliteStore) Maintain() error {
	_, err := s.DB.ExecContext(s.Ctx, maintenanceSQL)
	return err
}

// Clear() will drop all tables and recreate them.
// This is a destructive operation.
// Clear uses the same query as Create(), so it will also re-create the database
func (s *SqliteStore) Clear() error {
	_, err := s.DB.ExecContext(s.Ctx, createSQL)
	return err
}
