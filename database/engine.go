package database

import (
	"embed"
	"time"
)

type Engine interface {
	AfterOpen(dbh *DBHandle) error
	DSNSource() DataSource
	Driver() string
	MigrationFS() *embed.FS
}

// This interface is to expose a method to supply data to healthchecks.
type Observable interface {
	Stats(dbh *DBHandle) (any, error)
}

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
