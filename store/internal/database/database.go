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

// DSNGenerator is a function that returns a DSN string.
type DSNGenerator func() string

// StatementGenerator is a function that returns a prepared statement.
// The DBHandle holds a map of prepared statements, and will clean them up
// when closing.
type StatementGenerator func(ctx context.Context, db *sql.DB) (*sql.Stmt, error)

// MaintenanceFunction is a function that can be called periodically to
// perform maintenance on the database. It's passed the context and current
// database handle. Returning an error will stop the maintenance ticker.
type MaintenanceFunction func(ctx context.Context, db *sql.DB, t time.Time) error

const (
	SQLite DriverName = "sqlite3"
)

var (
	ErrDatabaseNotOpen      = errors.New("database not opened")
	ErrDatabaseAlreadyOpen  = errors.New("database already opened")
	ErrDatabaseClosed       = errors.New("database closed")
	ErrCantResetMaintenance = errors.New("can't reset maintenance ticker")
	ErrInvalidDuration      = errors.New("invalid duration for maintenance ticker")
	MinMaintenanceInterval  = 1 * time.Minute
)

type DBHandle[T comparable] struct {
	Ctx    context.Context
	DB     *sql.DB
	Driver DriverName
	DSN    DSNGenerator
	stmts  map[T]*sql.Stmt
	done   chan bool
	closed bool
}

type MaterialDB struct {
	DBHandle[string]
}

func NewDB(dsnF DSNGenerator) *MaterialDB {
	return &MaterialDB{
		DBHandle: DBHandle[string]{
			Driver: SQLite,
			DSN:    dsnF,
			stmts:  make(map[string]*sql.Stmt, 8),
		},
	}
}

func (s *DBHandle[T]) Open(ctx context.Context) error {
	s.Ctx = ctx
	// close this handle when the context is done
	context.AfterFunc(ctx, func() {
		s.Close()
	})
	db, err := sql.Open(string(s.Driver), s.DSN())
	if err != nil {
		return err
	}
	s.DB = db
	s.done = make(chan bool)
	return nil
}

// Pass a maintenance function and a duration to run it at.
// The maintenance function will be called with the context and the database handle.
// If the function returns an error, the ticker will be stopped.
// If the duration is 0 or less than a second, an error will be returned.
// It is possible to set up multiple maintenance functions.
// The Maintenance ticker will be stopped when the done channel receives a message or is closed.
// The done channel will be closed when this DBHandle is closed.
func (s *DBHandle[T]) Maintenance(d time.Duration, f MaintenanceFunction) error {
	if (d == 0) || (d < MinMaintenanceInterval) {
		return ErrInvalidDuration
	}
	ticker := time.NewTicker(d)
	go func() {
		defer ticker.Stop()
		for {
			select {
			case <-s.done:
				slog.Debug("DBHandle: maintenance ticker cancelled", "dsn", s.DSN())
				return
			case t := <-ticker.C:
				slog.Debug("DBHandle: maintenance ticker fired", "dsn", s.DSN(), "time", t)
				err := f(s.Ctx, s.DB, t)
				if err != nil {
					slog.Error("DBHandle: maintenance error, cancelling job", "dsn", s.DSN(), "error", err)
					return
				}
			}
		}
	}()
	return nil
}

func (s DBHandle[T]) Ping() error {
	if s.closed {
		return ErrDatabaseClosed
	}
	if s.DB == nil {
		return ErrDatabaseNotOpen
	}
	return s.DB.PingContext(s.Ctx)
}

func (s *DBHandle[T]) Statement(key T, generator StatementGenerator) (*sql.Stmt, error) {
	stmt, ok := s.stmts[key]
	if ok {
		return stmt, nil
	}
	var m sync.Mutex
	defer m.Unlock()
	m.Lock()
	if s.stmts == nil {
		s.stmts = make(map[T]*sql.Stmt, 8)
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
func (s *DBHandle[T]) Close() error {
	var m sync.Mutex
	m.Lock() // Aggressively lock this function
	defer m.Unlock()
	if s.DB == nil || s.closed {
		slog.Debug("db already closed, returning", "dsn", s.DSN())
		return nil
	}
	s.closed = true
	slog.Info("closing db", "dsn", s.DSN())
	// Possible problem here: if the context is not cancelled, we won't stop the ticker.
	close(s.done) // stop any maintenance tickers
	var errs []error
	for _, stmt := range s.stmts {
		if stmt != nil {
			err := stmt.Close()
			if err != nil {
				errs = append(errs, err)
			}
		}
	}
	clear(s.stmts)

	if s.DB != nil {
		err := s.DB.Close()
		if err != nil {
			errs = append(errs, err)
		}
		s.DB = nil
	}
	if len(errs) > 0 {
		slog.Warn("errors closing db", "dsn", s.DSN(), "errors", errs)
		return errors.Join(errs...)
	}
	return nil
}
