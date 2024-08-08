package database

import (
	"embed"
	"io/fs"
	"time"
)

// Interface for specific database implementations using DBHandle. SQL database
// handling is typically differentiated across 3 dimensions:
//
// 1. Configuration and DSN string generation
//
// 2. Open semantics (e.g. things you have to do when opening the connection)
//
// 3. Migrations and maintenance functions (which is typically where platform-specific
// SQL tends to show up)
//
// This interface (along with the supplemental/options interfaces below), provide
// the hooks to implement platform-specific behaviors in these 3 area. Application
// runtime  operations using DBHandle should utilize common SQL syntax, which is
// generally straightforward.
type Engine interface {
	// Provides DSN and basic configuration info
	DSNSource() DataSource
	// Driver Name
	Driver() string
	// Migrations directory, or nil if unsupported
	MigrationFS() fs.FS
}

// This interface is to expose a method to supply
// supplemental data (beyond what sql.Stats provides),
// in healthcheacks. The struct returned from this function
// will be added to the stats struct under the key 'engine'.
type Observable interface {
	Stats(dbh *DBHandle) (any, error)
}

// Interface for DB maintenance functions. If an engine implements this interface,
// its maintenance can be invoked on-demand from the `scrape` app.
// Not necessarily needed for setting up periodic maintenance.
type Maintainable interface {
	Maintain(dbh *DBHandle) error
}

// Provided for Engine implementations that want to do something
// right after the connection is opened.
type AfterOpenHook interface {
	AfterOpen(dbh *DBHandle) error
}

// If the engine provides this hook, it will be run before
// a migrate up operation is performed.
type BeforeMigrateUpHook interface {
	BeforeMigrateUp(dbh *DBHandle) error
}

// If the engine provides this hook, it should return a list of key/value
// pairs that will be inserted into the environment before a migration is run.
// These environment variables can be used in the migration scripts, and will
// be cleared after the migration is run.
type MigrationEnvSetter interface {
	MigrationEnv() []string
}

type BaseEngine struct {
	driver      string
	dsnSource   DataSource
	migrationFS fs.FS
}

// Provides a basic Engine implementation that can be used to build and test
// new DB implementations.
func NewEngine(driver string, dsnSource DataSource, migrationFS *embed.FS) BaseEngine {
	return BaseEngine{
		driver:      driver,
		dsnSource:   dsnSource,
		migrationFS: migrationFS,
	}
}

func (e BaseEngine) Driver() string {
	return e.driver
}

func (e BaseEngine) DSNSource() DataSource {
	return e.dsnSource
}

func (e BaseEngine) MigrationFS() fs.FS {
	return e.migrationFS
}

// BaseDataSource provides a basic DataSource implementation that's wrapped
// around a dsn string. The configiration-ey options (QueryTimeout, MaxConnections,
// ConnMaxLifetime) are all set to set to defaults that are generally useful for
// development and probably not appropriate for production.
//
// This implementation was initially provided for tests, but is currently unused and
// likely to be removed (or moved into a _test.go file) in the future.
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
