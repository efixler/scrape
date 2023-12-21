package sqlite

import (
	"context"
	"database/sql"
	_ "embed"
	"errors"
	"fmt"
	"io/fs"
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
	options := defaultOptions()
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

func dsn(filename string, options sqliteOptions) string {
	dsn := fmt.Sprintf(
		"file:%s?_busy_timeout=%d&_journal_mode=%s&_cache_size=%d&_sync=%s",
		filename,
		options.busyTimeout,
		options.journalMode,
		options.cacheSize,
		options.synchronous,
	)
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

func exists(fqn string) bool {
	if _, err := os.Stat(fqn); errors.Is(err, fs.ErrNotExist) {
		return false
	}
	return true
}
