package database

import (
	"context"
	"database/sql"
	"time"
)

// Execute an Exec with a timeout. If the timeout is less than or equal to 0, the query timeout is used.
// This function will return an error if the context is cancelled or the timeout is reached.
// This is safe to use, since sql.Result doesn't depend on anything happening after the query is executed.
func (s *DBHandle) ExecTimeout(timeout time.Duration, query string, args ...any) (sql.Result, error) {
	if timeout <= 0 {
		timeout = s.Engine.DSNSource().QueryTimeout()
	}
	ctx, cancel := context.WithTimeout(s.Ctx, timeout)
	defer cancel()
	return s.DB.ExecContext(ctx, query, args...)
}

// Experimental/Do not use: Execute a Query with a timeout.
// If the timeout is less than or equal to 0, the query timeout is used.
// The goal of this function was to provide a simple way for the caller to specify a timeout for a query,
// and encapsulate the timeout mechanics. However, since the Rows.Scan() is in the domain of the caller,
// but _is_ included in the timeout scope, we're forced here to keep the query active until the timeout
// is reached, which is a worse use of resources than a single long-running query.
// Leaving this here for reference (hope to get back to it at some point), but it's not recommended to use this function (or the similar QueryRowTimeout).
func (s *DBHandle) QueryTimeout(timeout time.Duration, query string, args ...any) (*sql.Rows, error) {
	if timeout <= 0 {
		timeout = s.Engine.DSNSource().QueryTimeout()
	}
	ctx, cancel := context.WithTimeout(s.Ctx, timeout)
	go func() {
		<-ctx.Done()
		cancel()
	}()
	return s.DB.QueryContext(ctx, query, args...)
}

// Experimental/Do not use: Execute a QueryRow with a timeout. If the timeout is less than or equal to 0, the query timeout is used.
func (s *DBHandle) QueryRowTimeout(timeout time.Duration, query string, args ...any) *sql.Row {
	if timeout <= 0 {
		timeout = s.Engine.DSNSource().QueryTimeout()
	}
	ctx, cancel := context.WithTimeout(s.Ctx, timeout)
	go func() {
		<-ctx.Done()
		cancel()
	}()
	return s.DB.QueryRowContext(ctx, query, args...)
}
