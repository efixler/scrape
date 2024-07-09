package database

import (
	"context"
	"database/sql"
	"errors"
	"testing"
	"time"

	_ "github.com/mattn/go-sqlite3"
)

func newDB(driver DriverName, dsnSource DataSource) *DBHandle {
	e := NewEngine(string(driver), dsnSource, nil)
	return New(e)
}

func TestMaintenanceRunsAndsStops(t *testing.T) {
	t.Parallel()
	oldMinInterval := MinMaintenanceInterval
	defer func() { MinMaintenanceInterval = oldMinInterval }()
	MinMaintenanceInterval = 1 * time.Millisecond
	count := 0
	mfunc := func(dbh *DBHandle) error {
		t.Logf("Maintenance ran at %s", time.Now())
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
	mfunc := func(dbh *DBHandle) error {
		t.Logf("Maintenance ran at %s", time.Now())
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
	engine := NewEngine(string(SQLite), NewDSN(":memory:"), nil)

	dbh := New(&engine)

	mdbh := &mockDBHandleForCloseTest{
		DBHandle: dbh,
	}

	// we don't want to cancel the context for this test
	err := mdbh.Open(context.Background())
	if err != nil {
		t.Fatalf("Error opening database: %s", err)
	}
	mf := func(dbh *DBHandle) error {
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

type testStmtKey int

func TestStatement(t *testing.T) {
	dbh := newDB(
		SQLite,
		NewDSN(
			":memory:",
			WithMaxConnections(1),
			WithConnMaxLifetime(-1),
		),
	)
	err := dbh.Open(context.Background())
	if err != nil {
		t.Fatalf("Error opening database: %s", err)
	}
	t.Cleanup(func() {
		dbh.Close()
	})
	genCallCount := 0

	gen := func(ctx context.Context, db *sql.DB) (*sql.Stmt, error) {
		genCallCount++
		return db.PrepareContext(ctx, "SELECT 1")
	}

	stmt1, err := dbh.Statement(testStmtKey(1), gen)
	if err != nil {
		t.Fatalf("Error preparing statement: %s", err)
	}
	defer stmt1.Close()
	if genCallCount != 1 {
		t.Errorf("Expected 1 generator call, got %d", genCallCount)
	}

	stmt2, err := dbh.Statement(testStmtKey(1), gen)
	if err != nil {
		t.Fatalf("Error retrieving preparing statement: %s", err)
	}
	defer stmt2.Close()
	if genCallCount != 1 {
		t.Errorf("Expected 1 generator call, got %d", genCallCount)
	}
	if stmt1 != stmt2 {
		t.Errorf("Expected same prepared statement, got a different one")
	}
	stmt3, err := dbh.Statement(1, gen)
	if err != nil {
		t.Fatalf("Error retrieving preparing statement: %s", err)
	}
	defer stmt3.Close()
	if genCallCount != 2 {
		t.Errorf("Expected 2 generator calls, got %d", genCallCount)
	}
	if stmt1 == stmt3 {
		t.Errorf("Expected different prepared statement, got the same one")
	}
	_, err = dbh.Statement(2, func(ctx context.Context, db *sql.DB) (*sql.Stmt, error) {
		return nil, errors.New("test error")
	})
	if err == nil {
		t.Errorf("Expected error on failed statement creation, got nil")
	}
	if len(dbh.stmts) != 2 {
		t.Errorf("Expected 2 statements in cache, got %d", len(dbh.stmts))
	}
}

func TestInvalidMaintenanceInterval(t *testing.T) {
	dbh := newDB(SQLite, NewDSN(":memory:", WithMaxConnections(1), WithConnMaxLifetime(-1)))
	err := dbh.Open(context.Background())
	if err != nil {
		t.Fatalf("Error opening database: %s", err)
	}
	defer dbh.Close()
	err = dbh.Maintenance(0, nil)
	if err != ErrInvalidDuration {
		t.Errorf("Expected error on invalid maintenance interval, got %v", err)
	}
}

type engineWithAfterOpen struct {
	Engine
	afterOpenCallCount int
}

func (e *engineWithAfterOpen) AfterOpen(dbh *DBHandle) error {
	e.afterOpenCallCount++
	return nil
}

func TestAfterOpenGetCalledWhenEngineImplements(t *testing.T) {
	ee := NewEngine(string(SQLite), NewDSN(":memory:"), nil)
	engine := &engineWithAfterOpen{Engine: ee}
	dbh := New(engine)
	err := dbh.Open(context.Background())
	if err != nil {
		t.Fatalf("Error opening database: %s", err)
	}
	defer dbh.Close()
	if engine.afterOpenCallCount != 1 {
		t.Errorf("Expected AfterOpen to be called once, got %d", engine.afterOpenCallCount)
	}
}

func TestCloseListenersInvoked(t *testing.T) {
	engine := NewEngine(string(SQLite), NewDSN(":memory:"), nil)
	dbh := New(engine)
	err := dbh.Open(context.Background())
	if err != nil {
		t.Fatalf("Error opening database: %s", err)
	}
	closeCount := 0
	closeF := func() {
		closeCount++
	}
	for i := 0; i < closeListenerCapacity; i++ {
		if err = dbh.AddCloseListener(closeF); err != nil {
			t.Fatalf("Error adding close listener: %s", err)
		}
	}
	if err := dbh.AddCloseListener(closeF); err != ErrCloseListenersFull {
		t.Errorf("Expected error on adding close listener when capacity reached, got %v", err)
	}
	dbh.Close()
	if closeCount != closeListenerCapacity {
		t.Errorf("Expected %d close listeners to be called, got %d", closeListenerCapacity, closeCount)
	}
	if err = dbh.AddCloseListener(closeF); err != ErrDatabaseClosed {
		t.Errorf("Expected error on adding close listener when already closed, got %v", err)
	}
}

func TestPing(t *testing.T) {
	dbh := newDB(SQLite, NewDSN(":memory:", WithMaxConnections(1), WithConnMaxLifetime(-1)))
	if err := dbh.Ping(); err != ErrDatabaseNotOpen {
		t.Errorf("Expected error pinging unopened database, got %v", err)
	}
	if err := dbh.Open(context.Background()); err != nil {
		t.Fatalf("Error opening database: %s", err)
	}
	if err := dbh.Ping(); err != nil {
		t.Errorf("Error pinging database: %s", err)
	}
	dbh.Close()
	if err := dbh.Ping(); err != ErrDatabaseClosed {
		t.Errorf("Expected error pinging closed database")
	}
}
