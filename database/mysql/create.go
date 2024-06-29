package mysql

import (
	"bytes"
	"embed"
	"errors"
	"text/template"

	"github.com/efixler/scrape/database"
)

//go:embed create.sql
var sql embed.FS

func (s *MySQL) createSQL() (string, error) {
	queryContent, _ := sql.ReadFile("create.sql")
	tmpl, err := template.New("create").Parse(string(queryContent))
	if err != nil {
		return "", err
	}

	// The connection we need to use for create must be schema-less so
	// that we can create the database, so we need to override that with
	// the default schema here.
	if s.config.TargetSchema == "" {
		return "", errors.New("can't create database, empty target schema")
	}
	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, s.config); err != nil {
		return "", err
	}
	return buf.String(), nil
}

func (s *MySQL) BeforeMigrateUp(dbh *database.DBHandle) error {
	q, err := s.createSQL()
	if err != nil {
		return err
	}
	_, err = dbh.DB.ExecContext(dbh.Ctx, q)
	return err
}
