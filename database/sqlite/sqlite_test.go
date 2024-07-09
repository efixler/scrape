package sqlite

import (
	"context"
	"errors"
	"testing"

	"github.com/efixler/scrape/database"
)

func TestInMemorySettingsWhenOpen(t *testing.T) {
	e, _ := New(InMemoryDB())
	dbh := database.New(e)
	err := dbh.Open(context.Background())
	if err != nil {
		t.Fatalf("Could not open in-memory db: %s", err)
	}
	defer dbh.Close()
	stats, err := dbh.Stats()
	if err != nil {
		t.Fatalf("Could not get stats: %s", err)
	}
	if stats.SQL.MaxOpenConnections != 1 {
		t.Errorf("Expected 1 MaxOpenConnections, got %d", stats.SQL.MaxOpenConnections)
	}
}

func TestErrorOnNew(t *testing.T) {
	_, err := New(InMemoryDB(), func(c *config) error {
		return errors.New("test error")
	})
	if err == nil {
		t.Errorf("Expected error, got nil")
	}
}

func TestMaintain(t *testing.T) {
	var tests = []struct {
		name        string
		sql         string
		expectError bool
	}{
		{
			name:        "good sql",
			sql:         "SELECT 1",
			expectError: false,
		},
		{
			name:        "bad sql",
			sql:         "SELECT * from urls",
			expectError: true,
		},
	}
	for _, tt := range tests {
		engine, err := New(InMemoryDB(), WithoutAutoCreate())
		if err != nil {
			t.Fatal(err)
		}
		context, cancel := context.WithCancel(context.Background())
		defer cancel()
		db := database.New(engine)
		err = db.Open(context)
		if err != nil {
			t.Fatalf("[%s] - unexpected error opening temp db: %v", tt.name, err)
		}
		maintenanceSQL = tt.sql
		err = engine.Maintain(db)
		if tt.expectError {
			if err == nil {
				t.Errorf("[%s] - expected error, got nil", tt.name)
			}
			continue
		}
		if err != nil {
			t.Fatalf("[%s] - expected no error, got %v", tt.name, err)
		}
	}
}
