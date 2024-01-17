package sqlite

import (
	"context"
	"testing"
	"time"
)

func TestStats(t *testing.T) {
	s, err := New(":memory:")
	if err != nil {
		t.Fatal(err)
	}
	db := s.(*SqliteStore)
	context, cancel := context.WithCancel(context.Background())
	defer cancel()
	err = db.Open(context)
	if err != nil {
		t.Fatal(err)
	}
	err = db.Create()
	if err != nil {
		t.Fatal(err)
	}
	stats, err := db.Stats()
	if err != nil {
		t.Fatal(err)
	}
	if stats.PageCount <= 0 {
		t.Errorf("Expected pages, got %d", stats.PageCount)
	}
	if stats.PageSize <= 0 {
		t.Errorf("Expected positive page size, got %d", stats.PageSize)
	}
	if stats.UnusedPages != 0 {
		t.Errorf("Expected no unused pages, got %d", stats.UnusedPages)
	}
	if stats.MaxPageCount <= 0 {
		t.Errorf("Expected positive max page count, got %d", stats.MaxPageCount)
	}
	if stats.DatabaseSizeMB() != 0 {
		t.Errorf("Expected 0MB database size, got %d", stats.DatabaseSizeMB())
	}
	if stats.SqliteVersion == "" {
		t.Errorf("Expected sqlite version, got empty string")
	}
	stats2, _ := db.Stats()
	if stats2.fetchTime != stats.fetchTime {
		t.Errorf("Expected stats fetch times to match, first: %v, second: %v", stats.fetchTime, stats2.fetchTime)
	}
}

func TestEmptyStatsIsExpired(t *testing.T) {
	var stats Stats
	if !stats.expired() {
		t.Errorf("Expected empty stats to be expired")
	}
}

func TestStatsIsExpired(t *testing.T) {
	fetchTime := time.Now().Add(-1 * minStatsInterval)
	stats := Stats{fetchTime: fetchTime}
	if !stats.expired() {
		t.Errorf("Expected stats to be expired")
	}
	stats.fetchTime = time.Now().Add(-1 * time.Second)
	if stats.expired() {
		t.Errorf("Expected stats to not be expired")
	}
}
