package database

import (
	"context"
	"testing"
)

func TestStats(t *testing.T) {
	dbh := newDB(SQLite, NewDSN(":memory:", WithMaxConnections(1), WithConnMaxLifetime(-1)))
	err := dbh.Open(context.Background())
	if err != nil {
		t.Fatalf("Error opening database: %s", err)
	}
	defer dbh.Close()
	stats, err := dbh.Stats()
	if err != nil {
		t.Fatalf("Error getting stats: %s", err)
	}
	if stats.SQL.MaxOpenConnections != 1 {
		t.Errorf("Expected 1 MaxOpenConnections, got %d", stats.SQL.MaxOpenConnections)
	}
}
