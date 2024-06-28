package database

import (
	"context"
	"database/sql"
	"time"
)

func (s *DBHandle) Exec(query string, args ...any) (sql.Result, error) {
	return s.ExecTimeout(s.engine.DSNSource().QueryTimeout(), query, args...)
}

func (s *DBHandle) ExecTimeout(timeout time.Duration, query string, args ...any) (sql.Result, error) {
	if timeout <= 0 {
		timeout = s.engine.DSNSource().QueryTimeout()
	}
	ctx, cancel := context.WithTimeout(s.Ctx, timeout)
	defer cancel()
	return s.DB.ExecContext(ctx, query, args...)
}

func (s *DBHandle) Query(query string, args ...any) (*sql.Rows, error) {
	return s.QueryTimeout(s.engine.DSNSource().QueryTimeout(), query, args...)
}

func (s *DBHandle) QueryTimeout(timeout time.Duration, query string, args ...any) (*sql.Rows, error) {
	if timeout <= 0 {
		timeout = s.engine.DSNSource().QueryTimeout()
	}
	ctx, cancel := context.WithTimeout(s.Ctx, timeout)
	go func() {
		<-ctx.Done()
		cancel()
	}()
	return s.DB.QueryContext(ctx, query, args...)
}

func (s *DBHandle) QueryRow(query string, args ...any) *sql.Row {
	return s.QueryRowTimeout(s.engine.DSNSource().QueryTimeout(), query, args...)
}

func (s *DBHandle) QueryRowTimeout(timeout time.Duration, query string, args ...any) *sql.Row {
	if timeout <= 0 {
		timeout = s.engine.DSNSource().QueryTimeout()
	}
	ctx, cancel := context.WithTimeout(s.Ctx, timeout)
	go func() {
		<-ctx.Done()
		cancel()
	}()
	return s.DB.QueryRowContext(ctx, query, args...)
}
