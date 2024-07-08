package sqlite

import (
	"context"
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
