package sqlite

import (
	"context"
	"testing"
)

func TestInMemorySettingsWhenOpen(t *testing.T) {
	s, err := New(InMemoryDB())
	if err != nil {
		t.Fatalf("Could not create new in-memory db: %s", err)
	}
	defer s.Close()
	err = s.Open(context.Background())
	if err != nil {
		t.Fatalf("Could not open in-memory db: %s", err)
	}
	stats := s.DB.Stats()
	if stats.MaxOpenConnections != 1 {
		t.Errorf("Expected 1 MaxOpenConnections, got %d", stats.MaxOpenConnections)
	}
}
