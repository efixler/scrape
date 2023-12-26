package sqlite

import "testing"

func TestInMemoryDSN(t *testing.T) {
	_, err := dbPath(InMemoryDBName)
	if err != ErrIsInMemory {
		t.Errorf("expected ErrIsInMemory, got %v", err)
	}
}
