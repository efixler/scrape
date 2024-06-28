package sqlite

import (
	"context"
	"database/sql"
	"embed"
	"time"

	"github.com/efixler/scrape/database"
	_ "github.com/mattn/go-sqlite3"
)

const SQLiteDriver = "sqlite3"

type SQLite struct {
	config config
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
var migrationFS embed.FS

func (s SQLite) MigrationFS() *embed.FS {
	return &migrationFS
}

// TODO: Trigger maintenance without this embed here (so it's not required)
//
//go:embed maintenance.sql
var maintenanceSQL string

func (s *SQLite) PostOpen(ctx context.Context, dbh *database.DBHandle) error {

	// SQLite will open even if the the DB file is not present, it will only fail later.
	// So, if the db hasn't been opened, check for the file here.
	// In Memory DBs must always be created
	if !s.config.databaseExists() && s.config.autoCreate() {
		if err := dbh.DoMigrateUp(); err != nil {
			return err
		}
	}
	dbh.Maintenance(
		24*time.Hour,
		func(ctx context.Context, db *sql.DB, tm time.Time) error {
			_, err := db.ExecContext(ctx, maintenanceSQL)
			return err
		},
	)
	return nil
}
