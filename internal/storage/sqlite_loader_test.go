//go:build !mysql

package storage

import (
	"context"
	_ "embed"
	"testing"

	"github.com/efixler/scrape/database"
	_ "github.com/mattn/go-sqlite3"
)

const (
	dbURL = "file::memory:?mode=memory&_busy_timeout=5000&_journal_mode=OFF&_cache_size=2000&_sync=NORMAL"
)

//go:embed sqlite/create.sql
var createSQL string

func getTestDatabase(t *testing.T) *SQLStorage {
	db := New(database.SQLite, dsn)
	err := db.Open(context.TODO())
	if err != nil {
		t.Fatalf("Error opening database: %v", err)
	}
	t.Cleanup(func() {
		t.Logf("Cleaning up SQLite test database")
		db.Close()
	})
	_, err = db.DB.Exec(createSQL)
	if err != nil {
		t.Fatalf("Error creating database: %v", err)
	}
	return db
}
