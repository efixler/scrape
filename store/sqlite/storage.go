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
	"time"

	"github.com/efixler/scrape/resource"
	"github.com/efixler/scrape/store"

	_ "github.com/mattn/go-sqlite3"
)

const (
	DEFAULT_DB_FILENAME  = "scrape_data/scrape.db"
	DEFAULT_BUSY_TIMEOUT = 5 * time.Second
	qInsert              = `REPLACE INTO urls (id, url, parsed_url, fetch_time, expires, metadata, content_text) VALUES (?, ?, ?, ?, ?, ?, ?)`
	qClear               = `DELETE FROM urls`
	qFetch               = `SELECT parsed_url, fetch_time, expires, metadata, content_text FROM urls WHERE id = ?`
	qDelete              = `DELETE FROM urls WHERE id = ?`
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
	deleteStmt  *sql.Stmt
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

func (s *sqliteStore) Store(uptr *store.StoredUrlData) (uint32, error) {
	uptr.AssertTimes() // modify the original with times if needed
	u := *uptr         // copy this so we don't modify the original below
	key := store.GetKey(u.Data.URL())
	contentText := u.Data.ContentText
	u.Data.ContentText = "" // make sure this is a copy
	if u.Data.ParsedUrl == nil {
		u.Data.ParsedUrl = u.Data.URL()
	}
	metadata, err := json.Marshal(u.Data)
	if err != nil {
		return 0, err
	}
	expires := time.Now().Add(*u.TTL).Unix()

	// (id, url, parsed_url, fetch_time, expires, metadata, content_text)
	values := []any{
		key,
		u.Data.URL().String(),
		u.Data.ParsedUrl.String(),
		u.FetchTime.Unix(),
		expires,
		string(metadata),
		contentText,
	}
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
	key := store.GetKey(url)
	db := dbs[s.dsn()]
	if s.fetchStmt == nil {
		stmt, err := db.PrepareContext(s.ctx, qFetch)
		if err != nil {
			return nil, err
		}
		s.fetchStmt = stmt
	}
	rows, err := s.fetchStmt.QueryContext(s.ctx, key)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	if !rows.Next() {
		return nil, nil
	}
	// parsed_url, fetch_time, expires, metadata, content_text
	var (
		parsedUrl   string
		fetchEpoch  int64
		expiryEpoch int64
		metadata    string
		contentText string
	)
	err = rows.Scan(&parsedUrl, &fetchEpoch, &expiryEpoch, &metadata, &contentText)
	if err != nil {
		return nil, err
	}
	exptime := time.Unix(expiryEpoch, 0)
	if time.Now().After(exptime) {
		// todo: delete this record
		return nil, nil
	}
	fetchTime := time.Unix(fetchEpoch, 0)
	ttl := exptime.Sub(fetchTime)
	page := &resource.WebPage{}
	page.ParsedUrl, err = nurl.Parse(parsedUrl)
	if err != nil {
		return nil, err
	}
	err = json.Unmarshal([]byte(metadata), page)
	if err != nil {
		return nil, err
	}
	page.ContentText = contentText
	sud := &store.StoredUrlData{
		Data:      *page,
		FetchTime: &fetchTime,
		TTL:       &ttl,
	}

	//fmt.Println(parsedUrl, fetchEpoch, expiryEpoch, metadata, contentText)
	return sud, nil
}

func (s *sqliteStore) Clear() error {
	db, err := s.openDB(s.dsn())
	if err != nil {
		return err
	}
	if _, err = db.ExecContext(s.ctx, qClear); err != nil {
		return err
	}
	return nil
}

func (s *sqliteStore) Delete(url *nurl.URL) (bool, error) {
	ustr := url.String()
	key := store.GetKey(ustr)
	db := dbs[s.dsn()]
	if s.deleteStmt == nil {
		stmt, err := db.PrepareContext(s.ctx, qDelete)
		if err != nil {
			return false, err
		}
		s.deleteStmt = stmt
	}
	result, err := s.deleteStmt.ExecContext(s.ctx, key)
	if err != nil {
		return false, err
	}
	rows, err := result.RowsAffected()
	if err != nil {
		return false, err
	}
	switch rows {
	case 0:
		return false, nil
	case 1:
		return true, nil
	default:
		return false, fmt.Errorf("expected 0 or 1 row affected, got %d", rows)
	}
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
