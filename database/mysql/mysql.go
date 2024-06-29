package mysql

import (
	"embed"

	"github.com/efixler/scrape/database"
)

const MySQLDriver = "mysql"

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
	s := &MySQL{
		config: config,
	}
	return s, nil
}

func (s MySQL) Driver() string {
	return MySQLDriver
}

func (s MySQL) DSNSource() database.DataSource {
	return s.config
}

//go:embed migrations/*.sql
var migrationFS embed.FS

func (s MySQL) MigrationFS() *embed.FS {
	return &migrationFS
}

func (s MySQL) AfterOpen(dbh *database.DBHandle) error {
	return nil
}

func (s MySQL) MigrationEnv() []string {
	return []string{"TargetSchema", s.config.Schema()}
}
