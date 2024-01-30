package mysql

import (
	_ "embed"
	"log/slog"
)

//go:embed create.sql
var createSQL string

func (s *MySQLStore) Create() error {
	_, err := s.DB.ExecContext(s.Ctx, createSQL)
	if err != nil {
		slog.Error("sqlite: error creating database", "error", err)
	}
	return err
}
