package database

import (
	"context"
	"database/sql"
	"embed"
	"errors"
	"testing"
	"time"

	_ "github.com/mattn/go-sqlite3"
)

type engine struct {
	driver      string
	dsnSource   DataSource
	migrationFS *embed.FS
}

func (e *engine) PostOpen(ctx context.Context, dbh *DBHandle) error {
	return nil
}

func (e *engine) DSNSource() DataSource {
	return e.dsnSource
}

func (e *engine) Driver() string {
	return e.driver
}

func (e *engine) MigrationFS() *embed.FS {
	return e.migrationFS
}

func newDB(driver DriverName, dsnSource DataSource) *DBHandle {
	e := &engine{
		driver:      string(driver),
		dsnSource:   dsnSource,
		migrationFS: nil,
	}

	return New(e)
}

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
	dbh := newDB(SQLite, NewDSN(":memory:", WithMaxConnections(1), WithConnMaxLifetime(-1)))
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
	dbh := newDB(SQLite, NewDSN(":memory:", WithMaxConnections(1), WithConnMaxLifetime(-1)))
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
	dbh := newDB(SQLite, NewDSN(":memory:", WithMaxConnections(1), WithConnMaxLifetime(-1)))
	err := dbh.Open(ctx)
	if err != nil {
		t.Fatalf("Error opening database: %s", err)
	}
	cancelF()
	time.Sleep(1 * time.Second)
	err = dbh.Ping()
	if err == nil {
		t.Errorf("Expected error pinging closed database")
	}
}

type mockDBHandleForCloseTest struct {
	*DBHandle
	maintCount int
}

func TestDBCloseExpectations(t *testing.T) {
	t.Parallel()

	dbh := New(&engine{
		driver:    string(SQLite),
		dsnSource: NewDSN(":memory:"),
	})

	mdbh := &mockDBHandleForCloseTest{
		DBHandle: dbh,
	}

	// we don't want to cancel the context for this test
	err := mdbh.Open(context.Background())
	if err != nil {
		t.Fatalf("Error opening database: %s", err)
	}
	mf := func(ctx context.Context, db *sql.DB, tm time.Time) error {
		mdbh.maintCount++
		return nil
	}
	err = mdbh.Maintenance(1*time.Minute, mf)
	if err != nil {
		t.Errorf("Error starting maintenance: %s", err)
	}
	err = mdbh.Close()
	if err != nil {
		t.Errorf("Error closing database: %s", err)
	}

	if !mdbh.DBHandle.closed {
		t.Errorf("Expected DBHandle to be closed")
	}

	if mdbh.DBHandle.DB != nil {
		t.Errorf("Expected DBHandle.DB to be nil")
	}

	select {
	case _, ok := <-mdbh.DBHandle.done:
		if ok {
			t.Errorf("done hannel is open, expected it to be closed")
		}
	default:
		t.Logf("No data received from the channel")
	}
	if len(mdbh.DBHandle.stmts) != 0 {
		t.Errorf("Expected stmts map to be empty, got %v", mdbh.DBHandle.stmts)
	}
}

func TestConnParams(t *testing.T) {
	t.Parallel()
	dbh := newDB(SQLite, NewDSN(":memory:", WithMaxConnections(1), WithConnMaxLifetime(-1)))
	err := dbh.Open(context.Background())
	if err != nil {
		t.Fatalf("Error opening database: %s", err)
	}
	if dbh.DB.Stats().MaxOpenConnections != 1 {
		t.Errorf("Expected 1 MaxOpenConnections, got %d", dbh.DB.Stats().MaxOpenConnections)
	}
	if dbh.DB.Stats().MaxIdleClosed != 0 {
		t.Errorf("Expected 0 MaxIdleClosed, got %d", dbh.DB.Stats().MaxIdleClosed)
	}
	if dbh.DB.Stats().MaxLifetimeClosed != 0 {
		t.Errorf("Expected 0 MaxLifetimeClosed, got %d", dbh.DB.Stats().MaxLifetimeClosed)
	}
}
