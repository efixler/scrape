package database

import (
	"context"
	"embed"
	"io/fs"
	"os"
	"testing"
)

//go:embed migration_test_embed/in_here/migrations/*.sql
var good_migrations_dir embed.FS

func TestExtractMigrationFS(t *testing.T) {
	var tests = []struct {
		name        string
		fs          fs.FS
		expectName  string
		expectError bool
	}{
		{
			name:        "good migrations dir (reference)",
			fs:          &good_migrations_dir,
			expectName:  "migration_test_embed/in_here/migrations",
			expectError: false,
		},
		{
			name:        "good migrations dir (value)",
			fs:          good_migrations_dir,
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

//go:embed migration_test_embed/in_here/migrations/*.sql
var dummy_migrations_dir embed.FS

func TestPrepareForMigration(t *testing.T) {
	tests := []struct {
		name        string
		migrationFS fs.FS
		env         []string
		expectError bool
	}{
		{
			name:        "no migration FS",
			migrationFS: nil,
			env:         nil,
			expectError: true,
		},
		{
			name:        "no extra env (nil)",
			migrationFS: dummy_migrations_dir,
			env:         nil,
			expectError: false,
		},
		{
			name:        "no extra env (empty slice)",
			migrationFS: dummy_migrations_dir,
			env:         []string{},
			expectError: false,
		},
		{
			name:        "odd number of extra env",
			migrationFS: dummy_migrations_dir,
			env:         []string{"one", "two", "three"},
			expectError: true,
		},
		{
			name:        "extra env",
			migrationFS: dummy_migrations_dir,
			env:         []string{"one", "two", "three", "four"},
			expectError: false,
		},
	}
	for _, tt := range tests {
		// Clear any relevant env variable to pin the unset-check
		for i := 0; i < len(tt.env); i += 2 {
			os.Setenv(tt.env[i], "")
		}

		e := NewEngine(
			"sqlite3",
			NewDataSource(":memory:"),
			tt.migrationFS,
		)
		db := New(e)
		clearF, err := db.prepareForMigration(tt.env...)
		if tt.expectError {
			if err == nil {
				t.Errorf("%s: expected error, got nil", tt.name)
			}
			continue
		}
		if err != nil {
			t.Fatalf("%s: expected no error, got %v", tt.name, err)
		}

		for i := 0; i < len(tt.env); i += 2 {
			val := os.Getenv(tt.env[i])
			if val != tt.env[i+1] {
				t.Errorf("%s: expected env %s to be %s, got %s", tt.name, tt.env[i], tt.env[i+1], val)
			}
		}
		clearF()
		for i := 0; i < len(tt.env); i += 2 {
			val := os.Getenv(tt.env[i])
			if val != "" {
				t.Errorf("%s: expected env %s to be unset, got %s", tt.name, tt.env[i], val)
			}
		}
	}
}
