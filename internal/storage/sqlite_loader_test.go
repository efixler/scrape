//go:build !mysql

package storage

import (
	"context"
	"embed"
	"testing"

	"github.com/efixler/scrape/database"
	_ "github.com/mattn/go-sqlite3"
)

const (
	dbURL = "file::memory:?mode=memory&_busy_timeout=5000&_journal_mode=OFF&_cache_size=2000&_sync=NORMAL"
)

//go:embed sqlite/migrations/*.sql
var migrationsFS embed.FS

func getTestDatabase(t *testing.T) *SQLStorage {
	db := New(database.SQLite, dsn)
	err := db.Open(context.TODO())
	if err != nil {
		t.Fatalf("Error opening database: %v", err)
	}
	t.Cleanup(func() {
		t.Logf("Cleaning up SQLite test database")
		// This is really just here to exercise the migration reset code path.
		// The SQLite test dbs will be destroyed when the connection is closed.
		if err := db.DoMigrateReset(migrationsFS, "sqlite/migrations"); err != nil {
			t.Logf("Error resetting SQLite test db: %v", err)
		}
		db.Close()
	})

	if err := db.DoMigrateUp(migrationsFS, "sqlite/migrations"); err != nil {
		t.Fatalf("Error creating SQLite test db via migration: %v", err)
	}

	return db
}
