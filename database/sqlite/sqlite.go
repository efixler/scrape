// DSN options and migration support for SQLite databases.
package sqlite

import (
	"embed"
	"io/fs"
	"time"

	"github.com/efixler/scrape/database"
	_ "github.com/mattn/go-sqlite3"
)

const SQLiteDriver = "sqlite3"

//go:embed migrations/*.sql
var migrationFS embed.FS

type SQLite struct {
	config config
	stats  *Stats
}

func MustNew(options ...Option) *SQLite {
	s, err := New(options...)
	if err != nil {
		panic(err)
	}
	return s
}

func New(options ...Option) (*SQLite, error) {
	c := &config{}
	Defaults()(c)
	for _, opt := range options {
		if err := opt(c); err != nil {
			return nil, err
		}
	}
	// If the migrationFS is not set, use the embedded migrations
	// This would make the embedded migrations in this package redundant --
	// it would be good to remove these to keep the engine implementation (reusable)
	// totally separate from migrations (not reusable().
	// However, that requires too many changes in too many places
	// (particularly in the tests) to be worth it right now, so leaving it hardwired/as-is.
	if c.migrationFS == nil {
		c.migrationFS = migrationFS
	}
	s := &SQLite{
		config: *c,
	}
	return s, nil
}

func (s SQLite) Driver() string {
	return SQLiteDriver
}

func (s SQLite) DSNSource() database.DataSource {
	return s.config
}

func (s *SQLite) MigrationFS() fs.FS {
	return s.config.migrationFS
}

func (s *SQLite) AfterOpen(dbh *database.DBHandle) error {

	// SQLite will open even if the the DB file is not present, it will only fail later.
	// So, if the db hasn't been opened, check for the file here.
	// In Memory DBs must always be created
	if !s.config.databaseExists() && s.config.autoCreate() {
		if err := dbh.MigrateUp(); err != nil {
			return err
		}
	}
	dbh.Maintenance(
		24*time.Hour,
		s.Maintain,
	)
	return nil
}

// TODO: Trigger maintenance without this embed here (so it's not required)
//
//go:embed maintenance.sql
var maintenanceSQL string

func (s *SQLite) Maintain(dbh *database.DBHandle) error {
	_, err := dbh.DB.ExecContext(dbh.Ctx, maintenanceSQL)
	return err
}
