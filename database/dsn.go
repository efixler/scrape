package database

import (
	"fmt"
	"time"
)

// DataSource is an interface that defines the options for a database connection.
type DataSource interface {
	// Loggable string representation of the options
	fmt.Stringer
	// Returns the DSN string for the options (not ever written to logs)
	DSN() string
	QueryTimeout() time.Duration
	MaxConnections() int
	ConnMaxLifetime() time.Duration
}

type dataSourceOption func(*DSNOptions)

// DSNOptions is a struct that implements the DataSourceOptions interface, a basic implementaion of DataSourceOptions.
type DSNOptions struct {
	DSNString       string
	queryTimeout    time.Duration
	maxConnections  int
	connMaxLifetime time.Duration
}

func (d DSNOptions) String() string {
	return d.DSNString
}

func (d DSNOptions) DSN() string {
	return d.DSNString
}

func (d DSNOptions) QueryTimeout() time.Duration {
	return d.queryTimeout
}

func (d DSNOptions) MaxConnections() int {
	return d.maxConnections
}

func (d DSNOptions) ConnMaxLifetime() time.Duration {
	return d.connMaxLifetime
}

func NewDSN(dsn string, options ...dataSourceOption) DSNOptions {
	d := &DSNOptions{
		DSNString: dsn,
	}
	for _, option := range options {
		option(d)
	}
	return *d
}

func WithQueryTimeout(timeout time.Duration) dataSourceOption {
	return func(d *DSNOptions) {
		d.queryTimeout = timeout
	}
}

func WithMaxConnections(maxConnections int) dataSourceOption {
	return func(d *DSNOptions) {
		d.maxConnections = maxConnections
	}
}

func WithConnMaxLifetime(lifetime time.Duration) dataSourceOption {
	return func(d *DSNOptions) {
		d.connMaxLifetime = lifetime
	}
}
