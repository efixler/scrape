package database

import (
	"context"
	"database/sql"
	"errors"
	"testing"
	"time"

	_ "github.com/mattn/go-sqlite3"
)

func TestMaintenanceRunsAndsStops(t *testing.T) {
	t.Parallel()
	oldMinInterval := MinMaintenanceInterval
	defer func() { MinMaintenanceInterval = oldMinInterval }()
	MinMaintenanceInterval = 1 * time.Millisecond
	count := 0
	mfunc := func(ctx context.Context, db *sql.DB, tm time.Time) error {
		t.Logf("Maintenance ran at %s", tm)
		count++
		return nil
	}
	ctx, cancelF := context.WithCancel(context.TODO())
	dbh := NewDB(func() string { return ":memory:" })
	err := dbh.Open(ctx)
	if err != nil {
		t.Fatalf("Error opening database: %s", err)
	}
	err = dbh.Maintenance(50*time.Millisecond, mfunc)
	if err != nil {
		t.Errorf("Error starting maintenance: %s", err)
	}
	time.Sleep(60 * time.Millisecond)
	cancelF()
	if count != 1 {
		t.Errorf("Maintenance: expected 1 iteration, got %d", count)
	}
	time.Sleep(100 * time.Millisecond)
	if count != 1 {
		t.Errorf("Maintenance not stopped: expected 1 iteration, got %d", count)
	}
}

func TestMaintenanceStopsOnError(t *testing.T) {
	t.Parallel()
	oldMinInterval := MinMaintenanceInterval
	defer func() { MinMaintenanceInterval = oldMinInterval }()
	MinMaintenanceInterval = 1 * time.Millisecond
	count := 0
	mfunc := func(ctx context.Context, db *sql.DB, tm time.Time) error {
		t.Logf("Maintenance ran at %s", tm)
		count++
		return errors.New("test error")
	}
	ctx, cancelF := context.WithCancel(context.Background())
	dbh := NewDB(func() string { return ":memory:" })
	err := dbh.Open(ctx)
	if err != nil {
		t.Fatalf("Error opening database: %s", err)
	}
	err = dbh.Maintenance(50*time.Millisecond, mfunc)
	if err != nil {
		t.Errorf("Error starting maintenance: %s", err)
	}
	time.Sleep(60 * time.Millisecond)
	if count == 0 {
		t.Errorf("Maintenance: expected at least 1 iteration, got %d", count)
	}
	ref := count
	time.Sleep(120 * time.Millisecond)
	if count != ref {
		t.Errorf("Maintenance not stopped: expected %d iterations, got %d", ref, count)
	}
	cancelF()
}

func TestDBClosedOnContextCancel(t *testing.T) {
	t.Parallel()
	ctx, cancelF := context.WithCancel(context.Background())
	dbh := NewDB(func() string { return ":memory:" })
	err := dbh.Open(ctx)
	if err != nil {
		t.Fatalf("Error opening database: %s", err)
	}
	cancelF()
	time.Sleep(1 * time.Millisecond)
	err = dbh.Ping()
	if err == nil {
		t.Errorf("Expected error pinging closed database")
	}
}
