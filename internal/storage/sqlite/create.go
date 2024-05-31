package sqlite

import (
	"context"
	"database/sql"
	"embed"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"time"

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

//go:embed migrations/*.sql
var migrationsFS embed.FS

func (s *Store) Reset() error {
	return s.DoMigrateReset(migrationsFS, "migrations")
}

func (s *Store) Migrate() error {
	return s.DoMigrateUp(migrationsFS, "migrations")
}

func (s *Store) MigrationStatus() error {
	return s.PrintMigrationStatus(migrationsFS, "migrations")
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

func (s *Store) Maintain() error {
	_, err := s.DB.ExecContext(s.Ctx, maintenanceSQL)
	return err
}

// // Clear() will drop all tables and recreate them.
// // This is a destructive operation.
// // Clear uses the same query as Create(), so it will also re-create the database
// func (s *Store) Clear() error {
// 	_, err := s.DB.ExecContext(s.Ctx, createSQL)
// 	return err
// }
