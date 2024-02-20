package mysql

import (
	_ "embed"
	"errors"
)

//go:embed create.sql
var createSQL string

func (s *Store) Create() error {
	_, err := s.DB.ExecContext(s.Ctx, createSQL)
	return err
}

func (s *Store) Clear() error {
	return s.Create()
}

func (s *Store) Maintain() error {
	return errors.New("mysql: maintain not implemented")
}
