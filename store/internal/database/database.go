package database

import (
	"context"
	"database/sql"
	"errors"
	"log/slog"
	"sync"
)

type DriverName string

type StatementGenerator func(ctx context.Context, db *sql.DB) (*sql.Stmt, error)

const (
	SQLite DriverName = "sqlite3"
)

var (
	ErrDatabaseNotOpen     = errors.New("database not opened")
	ErrDatabaseAlreadyOpen = errors.New("database already opened")
	ErrDatabaseClosed      = errors.New("database closed")
)

type DBHandle[T comparable] struct {
	Ctx    context.Context
	DB     *sql.DB
	Driver DriverName
	DSN    func() string
	stmts  map[T]*sql.Stmt
	closed bool
}

type MaterialDB struct {
	DBHandle[string]
}

func NewDB() *MaterialDB {
	return &MaterialDB{
		DBHandle: DBHandle[string]{
			Driver: SQLite,
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

// Close will be called when the context passed to Open() is cancelled
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
