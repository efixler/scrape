package database

import (
	"context"
	"embed"
)

type Engine interface {
	PostOpen(ctx context.Context, dbh *DBHandle) error
	DSNSource() DataSource
	Driver() string
	MigrationFS() *embed.FS
}

// type sqlite struct {
// 	*DBHandle[int]
// }

// func (s *sqlite) Open(ctx context.Context) error {
// 	err := s.DBHandle.Open(ctx)
// 	if err != nil {
// 		return err
// 	}
// 	// SQLite will open even if the the DB file is not present, it will only fail later.
// 	// So, if the db hasn't been opened, check for the file here.
// 	// In Memory DBs must always be created
// 	if !s.config.databaseExists() && s.config.autoCreate() {
// 		if err := s.Migrate(); err != nil {
// 			return err
// 		}
// 	}
// 	return s.DBHandle.Open(ctx)
// }
