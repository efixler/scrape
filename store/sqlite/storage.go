package sqlite

import (
	"context"
	"database/sql"
	_ "embed"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	nurl "net/url"
	"os"
	"time"

	"github.com/efixler/scrape/resource"
	"github.com/efixler/scrape/store"

	_ "github.com/mattn/go-sqlite3"
)

const (
	DEFAULT_BUSY_TIMEOUT = 5 * time.Second
	DEFAULT_JOURNAL_MODE = "WAL"
	DEFAULT_CACHE_SIZE   = -256000
	DEFAULT_SYNC         = "OFF"
	qStore               = `REPLACE INTO urls (id, url, parsed_url, fetch_time, expires, metadata, content_text) VALUES (?, ?, ?, ?, ?, ?, ?)`
	qClear               = `DELETE FROM urls; DELETE FROM id_map`
	qLookupId            = `SELECT canonical_id FROM id_map WHERE requested_id = ?`
	qStoreId             = `REPLACE INTO id_map (requested_id, canonical_id) VALUES (?, ?)`
	qClearId             = `DELETE FROM id_map where canonical_id = ?`
	qFetch               = `SELECT parsed_url, fetch_time, expires, metadata, content_text FROM urls WHERE id = ?`
	qDelete              = `DELETE FROM urls WHERE id = ?`
)

type stmtIndex int

const (
	_ stmtIndex = iota
	Store
	Clear
	LookupId
	StoreId
	ClearId
	Fetch
	Delete
)

var (
	dbs                map[string]*sql.DB
	ErrMappingNotFound = errors.New("id mapping not found")
	ErrStoreNotOpen    = errors.New("store not opened for this dsn")
)

func Factory(filename string) store.Factory {
	return func() (store.URLDataStore, error) {
		s := &sqliteStore{
			filename: filename,
			options:  defaultOptions(),
			stmts:    make(map[stmtIndex]*sql.Stmt, 8),
		}
		return s, nil
	}
}

type sqliteOptions struct {
	busyTimeout time.Duration
	journalMode string
	cacheSize   int
	synchronous string
}

type sqliteStore struct {
	ctx      context.Context
	filename string
	dsn      string
	options  sqliteOptions
	stmts    map[stmtIndex]*sql.Stmt
}

func defaultOptions() sqliteOptions {
	return sqliteOptions{
		busyTimeout: DEFAULT_BUSY_TIMEOUT,
		journalMode: DEFAULT_JOURNAL_MODE,
		cacheSize:   DEFAULT_CACHE_SIZE,
		synchronous: DEFAULT_SYNC,
	}
}

func (s *sqliteStore) Open(ctx context.Context) error {
	s.ctx = ctx
	fqn, err := dbPath(s.filename)
	if err != nil {
		return err
	}
	// SQLLite will open even if the the DB file is not present, it will only fail later.
	// So, we grab the db directly from the DSN map here, while we have the filename,
	// to make sure that we can print an informative error message if the file is missing.
	// This should only get called the first time a dsn is opened, as all instances of a struct
	// using the same dsn will share the same db.
	s.dsn = dsn(s.filename, s.options)
	_, ok := dbs[s.dsn]
	if ok {
		return nil
	}
	if _, err := os.Stat(fqn); os.IsNotExist(err) {
		return fmt.Errorf("database file %s does not exist", fqn)
	}
	db, err := sql.Open("sqlite3", s.dsn)
	if err != nil {
		log.Printf("error opening db: %v", err)
		return err
	}
	dbs[s.dsn] = db
	return nil
}

// Caller must close when done
func (s *sqliteStore) Close() error {
	var errs []error
	for _, stmt := range s.stmts {
		if stmt != nil {
			err := stmt.Close()
			if err != nil {
				errs = append(errs, err)
			}
		}
	}
	clear(s.stmts)
	if len(errs) > 0 {
		err := errors.Join(errs...)
		log.Printf("error closing sqlite: %v", err)
	}
	return nil
}

func (s *sqliteStore) Store(uptr *store.StoredUrlData) (uint64, error) {
	uptr.AssertTimes()             // modify the original with times if needed
	u := *uptr                     // copy this so we don't modify the original below
	key := store.Key(u.Data.URL()) // key is for the canonical URL
	contentText := u.Data.ContentText
	u.Data.ContentText = "" // make sure this is a copy
	if u.Data.RequestedURL == nil {
		u.Data.RequestedURL = u.Data.URL()
	}
	requestUrl := u.Data.RequestedURL.String()
	u.Data.RequestedURL = nil // make sure this is a copy
	fetchEpoch := u.Data.FetchTime.Unix()
	u.Data.FetchTime = nil
	metadata, err := json.Marshal(u.Data)
	if err != nil {
		return 0, err
	}
	expires := time.Now().Add(*u.TTL).Unix()

	// (id, url, parsed_url, fetch_time, expires, metadata, content_text)
	values := []any{
		key,
		u.Data.URL().String(),
		requestUrl,
		fetchEpoch,
		expires,
		string(metadata),
		contentText,
	}
	db := dbs[dsn(s.filename, s.options)]
	stmt, ok := s.stmts[Store]
	if !ok {
		stmt, err = db.PrepareContext(s.ctx, qStore)
		if err != nil {
			return 0, err
		}
		s.stmts[Store] = stmt
	}
	result, err := stmt.ExecContext(s.ctx, values...)
	if err != nil {
		return 0, err
	}
	// todo: this can fail silently
	err_id_map := s.storeIdMap(uptr.Data.RequestedURL, key)

	rows, err := result.RowsAffected()
	if err != nil {
		return 0, errors.Join(err, err_id_map)
	}
	if rows != 1 {
		return 0, errors.Join(fmt.Errorf("expected 1 row affected, got %d", rows), err_id_map)
	}
	return key, nil
}

func (s sqliteStore) storeIdMap(parsedUrl *nurl.URL, canonicalId uint64) error {
	// key := store.Key(parsedUrl)
	db := dbs[dsn(s.filename, s.options)]
	stmt, ok := s.stmts[StoreId]
	if !ok {
		var err error
		stmt, err = db.PrepareContext(s.ctx, qStoreId)
		if err != nil {
			return err
		}
		s.stmts[StoreId] = stmt
	}
	_, err := stmt.ExecContext(s.ctx, store.Key(parsedUrl), canonicalId)
	if err != nil {
		return err
	}
	return nil
}

func (s sqliteStore) Fetch(url *nurl.URL) (*store.StoredUrlData, error) {
	requested_key := store.Key(url)
	key, err := s.lookupId(requested_key)
	switch err {
	case ErrMappingNotFound:
		key = requested_key
	case nil: // do nothing
	default:
		return nil, err
	}
	db := dbs[dsn(s.filename, s.options)]
	stmt, ok := s.stmts[Fetch]
	if !ok {
		stmt, err = db.PrepareContext(s.ctx, qFetch)
		if err != nil {
			return nil, err
		}
		s.stmts[Fetch] = stmt
	}
	rows, err := stmt.QueryContext(s.ctx, key)
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
		// todo: delete this record (async, without leaking)
		return nil, store.ErrorResourceNotFound
	}
	fetchTime := time.Unix(fetchEpoch, 0).UTC()
	ttl := exptime.Sub(fetchTime)
	page := &resource.WebPage{FetchTime: &fetchTime}
	page.RequestedURL, err = nurl.Parse(parsedUrl)
	if err != nil {
		return nil, err
	}
	err = json.Unmarshal([]byte(metadata), page)
	if err != nil {
		return nil, err
	}
	page.ContentText = contentText
	sud := &store.StoredUrlData{
		Data: *page,
		TTL:  &ttl,
	}

	//fmt.Println(parsedUrl, fetchEpoch, expiryEpoch, metadata, contentText)
	return sud, nil
}

// Will search url_ids to see if there's a parent entry for this url.
func (s sqliteStore) lookupId(requested_id uint64) (uint64, error) {
	// key := store.Key(parsedUrl)
	db := dbs[dsn(s.filename, s.options)]
	stmt, ok := s.stmts[LookupId]
	if !ok {
		var err error
		stmt, err = db.PrepareContext(s.ctx, qLookupId)
		if err != nil {
			return 0, err
		}
		s.stmts[LookupId] = stmt
	}

	rows, err := stmt.QueryContext(s.ctx, requested_id)
	if err != nil {
		return 0, err
	}
	defer rows.Close()
	if !rows.Next() {
		return 0, ErrMappingNotFound
	}
	var lookupId uint64
	err = rows.Scan(&lookupId)
	if err != nil {
		return 0, err
	}
	return lookupId, nil
}

func (s *sqliteStore) Clear() error {
	db, ok := dbs[dsn(s.filename, s.options)]
	if !ok {
		return ErrStoreNotOpen
	}
	if _, err := db.ExecContext(s.ctx, qClear); err != nil {
		return err
	}
	return nil
}

func (s *sqliteStore) Delete(url *nurl.URL) (bool, error) {
	key := store.Key(url)
	db := dbs[dsn(s.filename, s.options)]
	stmt, ok := s.stmts[Delete]
	if !ok {
		stmt, err := db.PrepareContext(s.ctx, qDelete)
		if err != nil {
			return false, err
		}
		s.stmts[Delete] = stmt
	}
	result, err := stmt.ExecContext(s.ctx, key)
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

func init() {
	dbs = make(map[string]*sql.DB, 1)
}
