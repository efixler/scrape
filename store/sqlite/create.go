package sqlite

import (
	"context"
	"database/sql"
	_ "embed"
	"errors"
	"fmt"
	"os"
	"path/filepath"
)

const (
	DEFAULT_DB_FILENAME = "scrape_data/scrape.db"
)

var (
	ErrDatabaseExists = errors.New("database already exists")
)

//go:embed create.sql
var createSQL string

func CreateDB(ctx context.Context, filename string) error {
	fqn, err := dbPath(filename)
	if err != nil {
		return err
	}
	if exists(fqn) {
		return errors.Join(
			ErrDatabaseExists,
			fmt.Errorf("database file %s already exists, or the path can't be created", fqn),
		)
	}
	dir := filepath.Dir(fqn)
	if dh, _ := os.Stat(dir); dh == nil {
		err = os.MkdirAll(dir, 0775)
		if err != nil {
			return err
		}
	} else if !dh.IsDir() {
		return fmt.Errorf("path %s exists but is not a directory", dir)
	}
	options := DefaultOptions()
	options.synchronous = SQLITE_SYNC_NORMAL

	cdsn := dsn(fqn, options)
	// we will use a separate connection to create the db
	db, err := sql.Open("sqlite3", cdsn)
	if err != nil {
		return err
	}
	defer db.Close()
	_, err = db.ExecContext(ctx, createSQL)
	return err
}
