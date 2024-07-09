//go:build mysql

package mysql

import (
	"context"
	"testing"

	"github.com/efixler/scrape/database"
)

// Mysql integration tests assume a running mysql server
// at localhost:3306 with a root login and no password.

func testDatabaseForCreate(t *testing.T) *database.DBHandle {
	e, _ := New(
		Username("root"),
		Password(""),
		NetAddress("localhost:3306"),
		Schema("scrape_test"),
		ForMigration(),
	)
	dbh := database.New(e)

	// todo: enable alternate names when also creating
	// the database.
	t.Cleanup(func() {
		if err := dbh.MigrateReset(); err != nil {
			t.Errorf("Error resetting mysql test db %v: %v", "scrape_test", err)
		}
		if err := dbh.Close(); err != nil {
			t.Errorf("Error closing mysql database: %v", err)
		}
	})
	return dbh
}

// Test creating the db from scratch. The cleanup method above will migrate it down, deleting all
// of the tables, but keeping the db and the permissions.
func TestMigrate(t *testing.T) {
	db := testDatabaseForCreate(t)
	err := db.Open(context.Background())
	if err != nil {
		t.Errorf("Error opening database: %v", err)
	}
	err = db.MigrateUp()
	if err != nil {
		t.Errorf("Error creating database: %v", err)
	}
}

func TestBeforeMigrateUp(t *testing.T) {
	db := testDatabaseForCreate(t)
	err := db.Open(context.Background())
	if err != nil {
		t.Errorf("Error opening database: %v", err)
	}
	e, ok := db.Engine.(database.BeforeMigrateUpHook)
	if !ok {
		t.Fatalf("Engine does not implement BeforeMigrateUpHook")
	}

	err = e.BeforeMigrateUp(db)
	if err != nil {
		t.Errorf("Error before migrating up: %v", err)
	}
}
