package sqlite

import "testing"

func TestInMemoryDSN(t *testing.T) {
	_, err := dbPath(inMemoryDB)
	if err != ErrIsInMemory {
		t.Errorf("expected ErrIsInMemory, got %v", err)
	}
}
