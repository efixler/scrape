// DSN options and migration support for MySQL databases.
package mysql

import (
	"embed"
	"io/fs"

	"github.com/efixler/scrape/database"
)

const MySQLDriver = "mysql"

//go:embed migrations/*.sql
var migrationFS embed.FS

type MySQL struct {
	config Config
}

func New(options ...Option) (*MySQL, error) {
	config := defaultConfig()
	for _, opt := range options {
		if err := opt(&config); err != nil {
			return nil, err
		}
	}
	if config.migrationFS == nil {
		config.migrationFS = migrationFS
	}
	s := &MySQL{
		config: config,
	}
	return s, nil
}

func MustNew(options ...Option) *MySQL {
	s, err := New(options...)
	if err != nil {
		panic(err)
	}
	return s
}

func (s MySQL) Driver() string {
	return MySQLDriver
}

func (s MySQL) DSNSource() database.DataSource {
	return s.config
}

func (s MySQL) MigrationFS() fs.FS {
	return s.config.migrationFS
}

func (s MySQL) MigrationEnv() []string {
	return []string{"TargetSchema", s.config.Schema()}
}
