//go:build mysql

package mysql

import (
	"context"
	"testing"
)

// Mysql integration tests assume a running mysql server
// at localhost:3306 with a root login and no password.

func testStore() *Store {
	db, _ := New(
		Username("root"),
		Password(""),
		NetAddress("localhost:3306"),
	)
	return db // .(*Store)
}

func TestCreate(t *testing.T) {
	db := testStore()
	err := db.Open(context.Background())
	if err != nil {
		t.Errorf("Error opening database: %v", err)
	}
	defer db.Close()
	err = db.Create()
	if err != nil {
		t.Errorf("Error creating database: %v", err)
	}
}
