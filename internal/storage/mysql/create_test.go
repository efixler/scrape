//go:build mysql

package mysql

import (
	"context"
	"testing"
)

// Mysql integration tests assume a running mysql server
// at localhost:3306 with a root login and no password.

func testDatabaseForCreate(t *testing.T) *Store {
	db, _ := New(
		Username("root"),
		Password(""),
		NetAddress("localhost:3306"),
		Schema("scrape_test"),
		ForMigration(),
	)
	// todo: enable alternate names when also creating
	// the database.
	t.Cleanup(func() {
		if _, err := db.Exec("DROP DATABASE IF EXISTS scrape_test;"); err != nil {
			t.Logf("error dropping test mysql database: %q", err)
		}
		if err := db.Close(); err != nil {
			t.Errorf("Error closing mysql database: %v", err)
		}
	})
	return db // .(*Store)
}

func TestCreate(t *testing.T) {
	db := testDatabaseForCreate(t)
	err := db.Open(context.Background())
	if err != nil {
		t.Errorf("Error opening database: %v", err)
	}
	err = db.Create()
	if err != nil {
		t.Errorf("Error creating database: %v", err)
	}
}
