//go:build !mysql

package storage

import (
	"context"
	"embed"
	"testing"

	"github.com/efixler/scrape/database"
	_ "github.com/mattn/go-sqlite3"
	"github.com/pressly/goose/v3"
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
		db.Close()
	})
	goose.SetBaseFS(migrationsFS)
	if err := goose.SetDialect(string(goose.DialectSQLite3)); err != nil {
		t.Fatalf("Error setting dialect: %v", err)
	}
	if err := goose.Up(db.DB, "sqlite/migrations"); err != nil {
		t.Fatalf("Error creating SQLite test db via migration: %v", err)
	}
	return db
}
