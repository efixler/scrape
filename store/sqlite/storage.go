package sqlite

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	nurl "net/url"
	"os"
	"path/filepath"
	"scrape/store"
	"time"

	_ "github.com/mattn/go-sqlite3"
)

const (
	DEFAULT_DB_FILENAME  = "scrape_data/scrape.db"
	DEFAULT_BUSY_TIMEOUT = 5 * time.Second
	qInsert              = `REPLACE INTO urls (id, url, fetch_time, expires, metadata, content_text) VALUES (?, ?, ?, ?, ?, ?)`
)

var (
	dbs map[string]*sql.DB
)

type sqliteStore struct {
	ctx         context.Context
	filename    string
	busyTimeout time.Duration
	fetchStmt   *sql.Stmt
	storeStmt   *sql.Stmt
}

func Open(ctx context.Context, filename string) (*sqliteStore, error) {
	fqn, err := dbPath(filename)
	if err != nil {
		return nil, err
	}
	s := &sqliteStore{
		filename:    fqn,
		busyTimeout: DEFAULT_BUSY_TIMEOUT,
		ctx:         ctx,
	}
	dsn := s.dsn()
	_, err = s.openDB(dsn)
	if err != nil {
		return nil, err
	}
	return s, nil
}

// Caller must close when done
func (s *sqliteStore) Close() error {
	var errs []error
	for _, stmt := range []*sql.Stmt{s.fetchStmt, s.storeStmt} {
		if stmt != nil {
			err := stmt.Close()
			if err != nil {
				errs = append(errs, err)
			}
		}
	}
	if len(errs) > 0 {
		return errors.Join(errs...)
	}
	return nil
}

func (s *sqliteStore) Store(u *store.StoredUrlData) (uint32, error) {
	key := store.GetKey(u.Url)
	u.AssertTimes()
	metadata, err := json.Marshal(u.Metadata)
	if err != nil {
		return 0, err
	}
	expires := time.Now().Add(*u.TTL).Unix()
	// (id, url, fetch_time, expires, metadata, content_text)
	values := []any{key, u.Url.String(), u.FetchTime.Unix(), expires, string(metadata), u.ContentText}
	db := dbs[s.dsn()]
	if s.storeStmt == nil {
		stmt, err := db.PrepareContext(s.ctx, qInsert)
		if err != nil {
			return 0, err
		}
		s.storeStmt = stmt
	}
	result, err := s.storeStmt.ExecContext(s.ctx, values...)
	if err != nil {
		return 0, err
	}
	rows, err := result.RowsAffected()
	if err != nil {
		return 0, err
	}
	if rows != 1 {
		return 0, fmt.Errorf("expected 1 row affected, got %d", rows)
	}
	return key, nil
}

func (s sqliteStore) Fetch(url *nurl.URL) (*store.StoredUrlData, error) {
	return nil, nil
}

func (s *sqliteStore) openDB(dsn string) (*sql.DB, error) {
	db, ok := dbs[dsn]
	if ok {
		return db, nil
	}
	db, err := sql.Open("sqlite3", dsn)
	if err != nil {
		return nil, err
	}
	dbs[dsn] = db
	return db, nil
}

func (s sqliteStore) dsn() string {
	dsn := fmt.Sprintf("file:%s?_busy_timeout=%d", s.filename, s.busyTimeout)
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
		return filepath.Join(root, DEFAULT_DB_FILENAME), nil
	}
	_, err := os.Stat(filename)
	return filename, err
}

func init() {
	dbs = make(map[string]*sql.DB, 1)
}
