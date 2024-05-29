package mysql

import (
	"bytes"
	"embed"
	"errors"
	"text/template"

	"github.com/pressly/goose/v3"
)

//go:embed create.sql
var sql embed.FS

func (s *Store) createSQL() (string, error) {
	queryContent, _ := sql.ReadFile("create.sql")
	tmpl, err := template.New("create").Parse(string(queryContent))
	if err != nil {
		return "", err
	}
	conf := s.DSNSource.(Config)
	// The connection we need to use for create must be schema-less so
	// that we can create the database, so we need to override that with
	// the default schema here.
	if conf.TargetSchema == "" {
		return "", errors.New("can't create database, empty target schema")
	}
	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, conf); err != nil {
		return "", err
	}
	return buf.String(), nil
}

func (s *Store) create() error {
	q, err := s.createSQL()
	if err != nil {
		return err
	}
	_, err = s.DB.ExecContext(s.Ctx, q)
	return err
}

//go:embed migrations/*.sql
var migrationsFS embed.FS

func (s *Store) Migrate() error {
	if err := s.create(); err != nil {
		return err
	}
	goose.SetBaseFS(migrationsFS)
	if err := goose.SetDialect(string(goose.DialectMySQL)); err != nil {
		return err
	}
	if err := goose.Up(s.DB, "migrations"); err != nil {
		return err
	}
	return nil
}

func (s *Store) MigrationStatus() error {
	if err := goose.SetDialect(string(goose.DialectMySQL)); err != nil {
		return err
	}
	goose.SetBaseFS(migrationsFS)
	if err := goose.Status(s.DB, "migrations"); err != nil {
		return err
	}
	return nil
}

func (s *Store) Maintain() error {
	return errors.New("mysql: maintain not implemented")
}
