// DSN options and migration support for SQLite databases.
package sqlite

import (
	"embed"
	"time"

	"github.com/efixler/scrape/database"
	_ "github.com/mattn/go-sqlite3"
)

const SQLiteDriver = "sqlite3"

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

//go:embed migrations/*.sql
var MigrationFS embed.FS

func (s SQLite) MigrationFS() *embed.FS {
	return &MigrationFS
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
