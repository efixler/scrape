package sqlite

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"time"
)

const inMemoryDB = ":memory:"

var (
	ErrIsInMemory = errors.New("file path is in-memory DB (':memory:')")
)

type SqliteOptions struct {
	busyTimeout time.Duration
	journalMode string
	cacheSize   int
	synchronous string
}

func (o SqliteOptions) String() string {
	return fmt.Sprintf(
		"_busy_timeout=%d&_journal_mode=%s&_cache_size=%d&_sync=%s",
		o.busyTimeout,
		o.journalMode,
		o.cacheSize,
		o.synchronous,
	)
}

func DefaultOptions() SqliteOptions {
	return SqliteOptions{
		busyTimeout: DEFAULT_BUSY_TIMEOUT,
		journalMode: DEFAULT_JOURNAL_MODE,
		cacheSize:   DEFAULT_CACHE_SIZE,
		synchronous: DEFAULT_SYNC,
	}
}

func dsn(filename string, options SqliteOptions) string {
	dsn := fmt.Sprintf(
		"file:%s?%s",
		filename,
		options,
	)
	return dsn
}

// dbPath returns the path to the database file. If filename is empty,
// the path to the executable + the default path is returned.
// If filename is not empty filename is returned and its
// existence is checked.
func dbPath(filename string) (string, error) {
	switch filename {
	case inMemoryDB:
		return inMemoryDB, ErrIsInMemory
	case "":
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
