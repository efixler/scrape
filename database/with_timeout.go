package database

import (
	"context"
	"database/sql"
	"time"
)

func (s *DBHandle[T]) Exec(query string, args ...any) (sql.Result, error) {
	return s.ExecTimeout(s.DSNSource.QueryTimeout(), query, args...)
}

func (s *DBHandle[T]) ExecTimeout(timeout time.Duration, query string, args ...any) (sql.Result, error) {
	if timeout <= 0 {
		timeout = s.DSNSource.QueryTimeout()
	}
	ctx, cancel := context.WithTimeout(s.Ctx, timeout)
	defer cancel()
	return s.DB.ExecContext(ctx, query, args...)
}

func (s *DBHandle[T]) Query(query string, args ...any) (*sql.Rows, error) {
	return s.QueryTimeout(s.DSNSource.QueryTimeout(), query, args...)
}

func (s *DBHandle[T]) QueryTimeout(timeout time.Duration, query string, args ...any) (*sql.Rows, error) {
	if timeout <= 0 {
		timeout = s.DSNSource.QueryTimeout()
	}
	ctx, cancel := context.WithTimeout(s.Ctx, timeout)
	go func() {
		<-ctx.Done()
		cancel()
	}()
	return s.DB.QueryContext(ctx, query, args...)
}

func (s *DBHandle[T]) QueryRow(query string, args ...any) *sql.Row {
	return s.QueryRowTimeout(s.DSNSource.QueryTimeout(), query, args...)
}

func (s *DBHandle[T]) QueryRowTimeout(timeout time.Duration, query string, args ...any) *sql.Row {
	if timeout <= 0 {
		timeout = s.DSNSource.QueryTimeout()
	}
	ctx, cancel := context.WithTimeout(s.Ctx, timeout)
	go func() {
		<-ctx.Done()
		cancel()
	}()
	return s.DB.QueryRowContext(ctx, query, args...)
}
