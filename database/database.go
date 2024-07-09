package database

import (
	"context"
	"database/sql"
	"errors"
	"log/slog"
	"sync"
	"time"
)

type DriverName string

const (
	SQLite DriverName = "sqlite3"
	MySQL  DriverName = "mysql"
)

var (
	ErrDatabaseNotOpen      = errors.New("database not opened")
	ErrDatabaseAlreadyOpen  = errors.New("database already opened")
	ErrDatabaseClosed       = errors.New("database closed")
	ErrCantResetMaintenance = errors.New("can't reset maintenance ticker")
	ErrInvalidDuration      = errors.New("invalid duration for maintenance ticker")
	MinMaintenanceInterval  = 1 * time.Minute
)

// StatementGenerator is a function that returns a prepared statement.
// The DBHandle holds a map of prepared statements, and will clean them up
// when closing.
type StatementGenerator func(ctx context.Context, db *sql.DB) (*sql.Stmt, error)

// MaintenanceFunction is a function that can be called periodically to
// perform maintenance on the database. It's passed the context and current
// database handle. Returning an error will stop the maintenance ticker.
type MaintenanceFunction func(dbh *DBHandle) error

type DBHandle struct {
	Ctx    context.Context
	DB     *sql.DB
	Engine Engine
	stmts  map[any]*sql.Stmt
	done   chan bool
	closed bool
	mutex  *sync.Mutex
}

func New(engine Engine) *DBHandle {
	return &DBHandle{
		Engine: engine,
		stmts:  make(map[any]*sql.Stmt, 8),
	}
}

// Open the database handle with the given context. This handle will be closed if and
// when this context is cancelled. The context will also be used to prepare statements and
// as the basis for timeout-bound queries.
// Open-ing the connection will also apply the DataSource settings to the underlying DB
// connection *if* these settings are non-zero. Passing unset/zero values for these
// will inherit the driver defaults.
func (s *DBHandle) Open(ctx context.Context) error {
	if s.DB != nil {
		return ErrDatabaseAlreadyOpen
	}
	s.Ctx = ctx
	var err error
	s.DB, err = sql.Open(string(s.Engine.Driver()), s.Engine.DSNSource().DSN())
	slog.Info("opening db", "dsn", s.Engine.DSNSource, "driver", s.Engine.Driver())
	if err != nil {
		return err
	}
	// close this handle when the context is done
	context.AfterFunc(s.Ctx, func() {
		s.Close()
	})
	s.done = make(chan bool)
	s.mutex = &sync.Mutex{}
	if maxConns := s.Engine.DSNSource().MaxConnections(); maxConns != 0 {
		s.DB.SetMaxOpenConns(maxConns)
		s.DB.SetMaxIdleConns(maxConns)
	}
	if connMaxLifetime := s.Engine.DSNSource().ConnMaxLifetime(); connMaxLifetime != 0 {
		s.DB.SetConnMaxLifetime(connMaxLifetime)
	}
	if ae, ok := s.Engine.(AfterOpenHook); ok {
		return ae.AfterOpen(s)
	}
	return nil
}

// Convenience method to get the safe DSN string, with the password obscured.
// Implements fmt.Stringer interface.
func (s DBHandle) String() string {
	return s.Engine.DSNSource().String()
}

// Pass a maintenance function and a duration to run it at.
// The maintenance function will be called with the context and the database handle.
// If the function returns an error, the ticker will be stopped.
// If the duration is 0 or less than a second, an error will be returned.
// It is possible to set up multiple maintenance functions.
// The Maintenance ticker will be stopped when this DBHandle is closed,
// or with a StopMaintenance() call.
func (s *DBHandle) Maintenance(d time.Duration, f MaintenanceFunction) error {
	if (d == 0) || (d < MinMaintenanceInterval) {
		return ErrInvalidDuration
	}
	ticker := time.NewTicker(d)
	go func() {
		defer ticker.Stop()
		for {
			select {
			case <-s.done:
				slog.Debug("DBHandle: maintenance ticker cancelled", "dsn", s.Engine.DSNSource())
				return
			case t := <-ticker.C:
				slog.Debug("DBHandle: maintenance ticker fired", "dsn", s.Engine.DSNSource(), "time", t)
				err := f(s)
				if err != nil {
					slog.Error("DBHandle: maintenance error, cancelling job", "dsn", s.Engine.DSNSource(), "error", err)
					return
				}
			}
		}
	}()
	return nil
}

func (s *DBHandle) StopMaintenance() {
	close(s.done)
}

func (s *DBHandle) Ping() error {
	if s.closed {
		return ErrDatabaseClosed
	}
	if s.DB == nil {
		return ErrDatabaseNotOpen
	}
	return s.DB.PingContext(s.Ctx)
}

// Provides a means to cache prepared statements on a key. Use custom types
// on the key (e.g. how Context does it) to avoid collisions.
// The generator function will create the statement if it doesn't exist.
func (s *DBHandle) Statement(key any, generator StatementGenerator) (*sql.Stmt, error) {
	stmt, ok := s.stmts[key]
	if ok {
		return stmt, nil
	}
	s.mutex.Lock()
	defer s.mutex.Unlock()
	if s.stmts == nil {
		s.stmts = make(map[any]*sql.Stmt, 8)
	} else {
		stmt, ok = s.stmts[key]
		if ok {
			return stmt, nil
		}
	}
	stmt, err := generator(s.Ctx, s.DB)
	if err != nil {
		return nil, err
	}
	s.stmts[key] = stmt
	return stmt, nil
}

// Close will be called when the context passed to Open() is cancelled. It can
// also be called manually to release resources.
// It will close the database handle and any prepared statements, and stop any maintenance jobs.
func (s *DBHandle) Close() error {
	if s.closed || (s.DB == nil) {
		slog.Debug("db already closed, returning", "dsn", s.Engine.DSNSource())
		return nil
	}
	s.mutex.Lock()
	defer s.mutex.Unlock()
	if s.closed || (s.DB == nil) {
		return nil
	}
	s.closed = true
	slog.Info("closing db", "dsn", s.Engine.DSNSource())
	s.StopMaintenance()
	var errs []error
	for _, stmt := range s.stmts {
		if stmt != nil {
			if err := stmt.Close(); err != nil {
				errs = append(errs, err)
			}
		}
	}
	clear(s.stmts)

	if s.DB != nil {
		if err := s.DB.Close(); err != nil {
			errs = append(errs, err)
		}
		s.DB = nil
	}

	if len(errs) > 0 {
		slog.Warn("errors closing db", "dsn", s.Engine.DSNSource(), "errors", errs)
		return errors.Join(errs...)
	}
	return nil
}

type stats struct {
	SQL    sql.DBStats `json:"sql"`
	Engine any         `json:"engine,omitempty"`
}

func (s *DBHandle) Stats() (*stats, error) {
	if s.DB == nil {
		return nil, ErrDatabaseNotOpen
	}
	stats := &stats{
		SQL: s.DB.Stats(),
	}

	if observableEngine, ok := s.Engine.(Observable); ok {
		var err error
		stats.Engine, err = observableEngine.Stats(s)
		if err != nil {
			slog.Error("error getting engine stats", "error", err)
		}
	}
	return stats, nil
}
