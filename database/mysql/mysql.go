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

//go:embed migrations/*.sql
var MigrationFS embed.FS

func (s MySQL) MigrationFS() *embed.FS {
	return &MigrationFS
}

func (s MySQL) MigrationEnv() []string {
	return []string{"TargetSchema", s.config.Schema()}
}
