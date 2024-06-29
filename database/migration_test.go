package database

import (
	"context"
	"embed"
	"testing"
)

//go:embed migration_test_embed/in_here/migrations/*.sql
var good_migrations_dir embed.FS

func TestExtractMigrationFS(t *testing.T) {
	var tests = []struct {
		name        string
		fs          *embed.FS
		expectName  string
		expectError bool
	}{
		{
			name:        "good migrations dir",
			fs:          &good_migrations_dir,
			expectName:  "migration_test_embed/in_here/migrations",
			expectError: false,
		},
		{
			name:        "nil migrations dir",
			fs:          nil,
			expectName:  "",
			expectError: true,
		},
	}

	for _, tt := range tests {
		d, name, err := extractMigrationFS(tt.fs)
		if tt.expectError {
			if err == nil {
				t.Errorf("%s: expected error, got nil", tt.name)
			}
		} else {
			if err != nil {
				t.Fatalf("%s: expected no error, got %v", tt.name, err)
			}
			if name != tt.expectName {
				t.Errorf("%s: expected name %s, got %s", tt.name, tt.expectName, name)
			}
			if d == nil {
				t.Errorf("%s: expected an fs, got nil", tt.name)
			}
		}
	}
}

func TestMigration(t *testing.T) {
	dsn := NewDataSource(":memory:")
	e := NewEngine(string(SQLite), dsn, &good_migrations_dir)
	dbh := New(e)
	err := dbh.Open(context.Background())
	if err != nil {
		t.Fatalf("Error opening database: %v", err)
	}
	t.Cleanup(func() {
		dbh.Close()
	})
	err = dbh.MigrateUp()
	if err != nil {
		t.Fatalf("Error migrating up: %v", err)
	}
	err = dbh.MigrateReset()
	if err != nil {
		t.Fatalf("Error migrating reset: %v", err)
	}
}
