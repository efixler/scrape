//go:build !mysql

package storage

import (
	"context"
	"testing"

	"github.com/efixler/scrape/database"
	"github.com/efixler/scrape/database/sqlite"
	_ "github.com/mattn/go-sqlite3"
)

const (
	dbURL = "file::memory:?mode=memory&_busy_timeout=5000&_journal_mode=OFF&_cache_size=2000&_sync=NORMAL"
)

// //go:embed ../../database/sqlite/migrations/*.sql
// var migrationsFS embed.FS

func getTestDatabase(t *testing.T) *URLDataStore {
	e := database.NewEngine(
		string(database.SQLite),
		dsn,
		&sqlite.MigrationFS,
	)
	db := database.New(e)
	urlStore := NewURLDataStore(db)
	// TODO: important - the url store is managing db opens and
	// closes, and it must not
	err := urlStore.Open(context.TODO())
	if err != nil {
		t.Fatalf("Error opening database: %v", err)
	}
	t.Cleanup(func() {
		t.Logf("Cleaning up SQLite test database")
		// This is really just here to exercise the migration reset code path.
		// The SQLite test dbs will be destroyed when the connection is closed.
		if err := db.MigrateReset(); err != nil {
			t.Logf("Error resetting SQLite test db: %v", err)
		}
		urlStore.Close()
	})

	if err := db.MigrateUp(); err != nil {
		t.Fatalf("Error creating SQLite test db via migration: %v", err)
	}

	return urlStore
}
