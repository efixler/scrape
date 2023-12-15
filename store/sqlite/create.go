package sqlite

import (
	"context"
	"database/sql"
	_ "embed"
	"fmt"
	"os"
	"path/filepath"
)

const (
	DEFAULT_DB_FILENAME = "scrape_data/scrape.db"
)

//go:embed create.sql
var createSQL string

func CreateDB(ctx context.Context, filename string) error {
	fqn, err := dbPath(filename)
	if err != nil {
		return err
	}
	if _, err = os.Stat(fqn); !os.IsNotExist(err) {
		return fmt.Errorf("database file %s already exists, or the path can't be created", fqn)
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

	cdsn := dsn(fqn, defaultOptions())
	// we will use a separate connection to create the db
	db, err := sql.Open("sqlite3", cdsn)
	if err != nil {
		return err
	}
	defer db.Close()
	_, err = db.ExecContext(ctx, createSQL)
	return err
}

func dsn(filename string, options sqliteOptions) string {
	dsn := fmt.Sprintf("file:%s?_busy_timeout=%d", filename, options.busyTimeout)
	return dsn
}

// dbPath returns the path to the database file. If filename is empty,
// the path to the executable + the default path is returned.
// If filename is not empty filename is returned and its
// existence is checked.
func dbPath(filename string) (string, error) {
	if filename == "" {
		root, err := os.Executable()
		if err != nil {
			return "", err
		}
		root, err = filepath.Abs(root)
		if err != nil {
			return "", err
		}
		filename = filepath.Join(root, DEFAULT_DB_FILENAME)
	}
	return filepath.Abs(filename)
}
