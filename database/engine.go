package database

import (
	"embed"
	"time"
)

// Interface for database implementations usin DBHandle
type Engine interface {
	// Called after the connection is opened
	AfterOpen(dbh *DBHandle) error
	// Provides DSN and basic configuratiin info
	DSNSource() DataSource
	// Driver Name
	Driver() string
	// Migrations directory, or nil if unsupported
	MigrationFS() *embed.FS
}

// This interface is to expose a method to supply
// supplemental data (beyond what sql.Stats provides),
// in healthcheacks. The struct returned from this function
// will be added to the stats struct under the key 'engine'.
type Observable interface {
	Stats(dbh *DBHandle) (any, error)
}

// Interface for DB maintenance functions. Required for invoking
// maintenance on-demand, not necessarily needed for setting up
// periodic maintenance.
type Maintainable interface {
	Maintain(dbh *DBHandle) error
}

type BeforeMigrateUpHook interface {
	BeforeMigrateUp(dbh *DBHandle) error
}

type MigrationEnvSetter interface {
	MigrationEnv() []string
}

// TODO: Provide a base engine implementation that can be embedded in other engines

type BaseEngine struct {
	driver      string
	dsnSource   DataSource
	migrationFS *embed.FS
}

func NewEngine(driver string, dsnSource DataSource, migrationFS *embed.FS) BaseEngine {
	return BaseEngine{
		driver:      driver,
		dsnSource:   dsnSource,
		migrationFS: migrationFS,
	}
}

func (e BaseEngine) AfterOpen(dbh *DBHandle) error {
	return nil
}

func (e BaseEngine) Driver() string {
	return e.driver
}

func (e BaseEngine) DSNSource() DataSource {
	return e.dsnSource
}

func (e BaseEngine) MigrationFS() *embed.FS {
	return e.migrationFS
}

type BaseDataSource string

func NewDataSource(dsn string) BaseDataSource {
	return BaseDataSource(dsn)
}

func (d BaseDataSource) String() string {
	return string(d)
}

func (d BaseDataSource) DSN() string {
	return string(d)
}

func (d BaseDataSource) QueryTimeout() time.Duration {
	return 10 * time.Second
}

func (d BaseDataSource) MaxConnections() int {
	return 1
}

func (d BaseDataSource) ConnMaxLifetime() time.Duration {
	return 0
}
