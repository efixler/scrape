/*
This is the implementation of the store.URLDataStore interface for sqlite.

Use New() to make a new sqlite storage instance.
  - You *must* call Open()
  - The DB will be closed when the context passed to Open() is cancelled.
  - Concurrent usage OK
  - In-Memory DBs are supported
  - The DB will be created if it doesn't exist
*/
package sqlite

import (
	"context"
	_ "embed"
	"time"

	"github.com/efixler/scrape/database"
	"github.com/efixler/scrape/internal/storage"
	"github.com/efixler/scrape/store"

	_ "github.com/mattn/go-sqlite3"
)

// Store is the sqlite implementation of the store.URLDataStore interface.
// It relies on storage.SQLStorage for most of the actual database operations,
// and mainly handles configuration and initialization.
type Store struct {
	*storage.SQLStorage
	config config
	stats  *Stats
}

// Returns the factory function that can be used to instantiate a sqlite store
// in the cases where either creation should be delayed or where the caller may
// want to instantiate multiple stores with the same configuration.
func Factory(options ...option) store.Factory {
	return func() (store.URLDataStore, error) {
		return New(options...)
	}
}

func New(options ...option) (store.URLDataStore, error) {
	c := &config{}
	Defaults()(c)
	for _, opt := range options {
		if err := opt(c); err != nil {
			return nil, err
		}
	}
	s := &Store{
		SQLStorage: storage.New(database.SQLite, c),
		config:     *c,
	}
	return s, nil
}

// Opens the database, creating it if it doesn't exist.
// The passed contexts will be used for query preparation, and to
// close the database when the context is cancelled.
func (s *Store) Open(ctx context.Context) error {
	err := s.DBHandle.Open(ctx)
	if err != nil {
		return err
	}
	// SQLite will open even if the the DB file is not present, it will only fail later.
	// So, if the db hasn't been opened, check for the file here.
	// In Memory DBs must always be created
	inMemory := s.config.IsInMemory()
	needsCreate := inMemory || !exists(s.config.filename)
	if needsCreate {
		if err := s.Create(); err != nil {
			return err
		}
	}
	if inMemory {
		// Unfortunately, SQLite in-memory DBs are bound to a single connection.
		s.DB.SetMaxOpenConns(1)
		s.DB.SetMaxIdleConns(1)
		s.DB.SetConnMaxLifetime(-1)
	}
	s.Maintenance(24*time.Hour, maintain)
	return nil
}
