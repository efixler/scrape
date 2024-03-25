package database

import (
	"testing"
	"time"
)

// There's no (straightforward) way to test actual set values of these in the SQL connection so
// just test the correctness of the DSN struct
func TestDSNSettings(t *testing.T) {
	tests := []struct {
		name        string
		dsn         string
		maxConns    int
		timeout     time.Duration
		maxLifetime time.Duration
	}{
		{
			name:        "No options",
			dsn:         ":memory:",
			maxConns:    0,
			timeout:     0,
			maxLifetime: 0,
		},
		{
			name:        "With options",
			dsn:         ":memory:",
			maxConns:    1,
			timeout:     10 * time.Second,
			maxLifetime: 5 * time.Minute,
		},
	}
	for _, test := range tests {
		dsn := NewDSN(test.dsn,
			WithMaxConnections(test.maxConns),
			WithQueryTimeout(test.timeout),
			WithConnMaxLifetime(test.maxLifetime))
		if dsn.QueryTimeout() != test.timeout {
			t.Errorf("QueryTimeout not set correctly: %v", dsn.QueryTimeout())
		}
		if dsn.MaxConnections() != test.maxConns {
			t.Errorf("MaxConnections not set correctly: %v", dsn.MaxConnections())
		}
		if dsn.ConnMaxLifetime() != test.maxLifetime {
			t.Errorf("ConnMaxLifetime not set correctly: %v", dsn.ConnMaxLifetime())
		}
	}
}

func TestDSNZeros(t *testing.T) {
	dsn := NewDSN(":memory:")
	if dsn.QueryTimeout() != 0 {
		t.Errorf("Expected QueryTimeout to be 0, got %v", dsn.QueryTimeout())
	}
	if dsn.MaxConnections() != 0 {
		t.Errorf("Expected MaxConnections to be 0, got %v", dsn.MaxConnections())
	}
	if dsn.ConnMaxLifetime() != 0 {
		t.Errorf("Expected ConnMaxLifetime to be 0, got %v", dsn.ConnMaxLifetime())
	}
}
