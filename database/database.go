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
type MaintenanceFunction func(ctx context.Context, db *sql.DB, t time.Time) error

type DBHandle[T comparable] struct {
	Ctx       context.Context
	DB        *sql.DB
	Driver    DriverName
	DSNSource DataSource
	stmts     map[T]*sql.Stmt
	done      chan bool
	closed    bool
	mutex     *sync.Mutex
}

// Open the database handle with the given context. This handle will be closed if and
// when this context is cancelled. The context will also be used to prepare statements and
// as the basis for timeout-bound queries.
// Open-ing the connection will also apply the DataSource settings to the underlying DB
// connection *if* these settings are non-zero. Passing unset/zero values for these
// will inherit the driver defaults.
func (s *DBHandle[T]) Open(ctx context.Context) error {
	if s.DB != nil {
		return ErrDatabaseAlreadyOpen
	}
	s.Ctx = ctx
	var err error
	s.DB, err = sql.Open(string(s.Driver), s.DSNSource.DSN())
	slog.Info("opening db", "dsn", s.DSNSource, "driver", s.Driver)
	if err != nil {
		return err
	}
	// close this handle when the context is done
	context.AfterFunc(s.Ctx, func() {
		s.Close()
	})
	s.done = make(chan bool)
	s.mutex = &sync.Mutex{}
	if maxConns := s.DSNSource.MaxConnections(); maxConns != 0 {
		s.DB.SetMaxOpenConns(maxConns)
		s.DB.SetMaxIdleConns(maxConns)
	}
	if connMaxLifetime := s.DSNSource.ConnMaxLifetime(); connMaxLifetime != 0 {
		s.DB.SetConnMaxLifetime(connMaxLifetime)
	}

	return nil
}

// Convenience method to get the safe DSN string, with the password obscured.
// Implements fmt.Stringer interface.
func (s DBHandle[T]) String() string {
	return s.DSNSource.String()
}

// Pass a maintenance function and a duration to run it at.
// The maintenance function will be called with the context and the database handle.
// If the function returns an error, the ticker will be stopped.
// If the duration is 0 or less than a second, an error will be returned.
// It is possible to set up multiple maintenance functions.
// The Maintenance ticker will be stopped when this DBHandle is closed,
// or with a StopMaintenance() call.
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
				slog.Debug("DBHandle: maintenance ticker cancelled", "dsn", s.DSNSource)
				return
			case t := <-ticker.C:
				slog.Debug("DBHandle: maintenance ticker fired", "dsn", s.DSNSource, "time", t)
				err := f(s.Ctx, s.DB, t)
				if err != nil {
					slog.Error("DBHandle: maintenance error, cancelling job", "dsn", s.DSNSource, "error", err)
					return
				}
			}
		}
	}()
	return nil
}

func (s *DBHandle[T]) StopMaintenance() {
	close(s.done)
}

func (s *DBHandle[T]) Ping() error {
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
	s.mutex.Lock()
	defer s.mutex.Unlock()
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
	if s.closed || (s.DB == nil) {
		slog.Debug("db already closed, returning", "dsn", s.DSNSource)
		return nil
	}
	s.mutex.Lock()
	defer s.mutex.Unlock()
	if s.closed || (s.DB == nil) {
		return nil
	}
	s.closed = true
	slog.Info("closing db", "dsn", s.DSNSource)
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
		slog.Warn("errors closing db", "dsn", s.DSNSource, "errors", errs)
		return errors.Join(errs...)
	}
	return nil
}
