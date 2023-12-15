package sqlite

import (
	"context"
	"database/sql"
	_ "embed"
	"encoding/json"
	"errors"
	"fmt"
	nurl "net/url"
	"os"
	"time"

	"github.com/efixler/scrape/resource"
	"github.com/efixler/scrape/store"

	_ "github.com/mattn/go-sqlite3"
)

const (
	DEFAULT_BUSY_TIMEOUT = 5 * time.Second
	qStore               = `REPLACE INTO urls (id, url, parsed_url, fetch_time, expires, metadata, content_text) VALUES (?, ?, ?, ?, ?, ?, ?)`
	qClear               = `DELETE FROM urls`
	qLookupId            = `SELECT url_id FROM id_map WHERE parsed_url_id = ?`
	qStoreId             = `REPLACE INTO id_map (parsed_url_id, url_id) VALUES (?, ?)`
	qClearId             = `DELETE FROM id_map where url_id = ?`
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

type sqliteOptions struct {
	busyTimeout time.Duration
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
	}
}

func Open(ctx context.Context, filename string) (*sqliteStore, error) {
	fqn, err := dbPath(filename)
	if err != nil {
		return nil, err
	}
	s := &sqliteStore{
		filename: fqn,
		options:  defaultOptions(),
		ctx:      ctx,
		stmts:    make(map[stmtIndex]*sql.Stmt, 8),
	}
	// SQLLite will open even if the the DB file is not present, it will only fail later.
	// So, we grab the db directly from the DSN map here, while we have the filename,
	// to make sure that we can print an informative error message if the file is missing.
	// This should only get called the first time a dsn is opened, as all instances of a struct
	// using the same dsn will share the same db.
	s.dsn = dsn(s.filename, s.options)
	_, ok := dbs[s.dsn]
	if ok {
		return s, nil
	}
	if _, err := os.Stat(fqn); os.IsNotExist(err) {
		return nil, fmt.Errorf("database file %s does not exist", fqn)
	}
	db, err := sql.Open("sqlite3", s.dsn)
	if err != nil {
		fmt.Println("error opening db", err)
		return nil, err
	}
	dbs[s.dsn] = db
	return s, nil
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
	err_id_map := s.storeIdMap(u.Data.ParsedUrl, key)
	rows, err := result.RowsAffected()
	if err != nil {
		return 0, errors.Join(err, err_id_map)
	}
	if rows != 1 {
		return 0, errors.Join(fmt.Errorf("expected 1 row affected, got %d", rows), err_id_map)
	}
	return key, nil
}

func (s sqliteStore) storeIdMap(parsedUrl *nurl.URL, canonicalId uint32) error {
	key := store.GetKey(parsedUrl)
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
	_, err := stmt.ExecContext(s.ctx, key, canonicalId)
	if err != nil {
		return err
	}
	return nil
}

func (s sqliteStore) Fetch(url *nurl.URL) (*store.StoredUrlData, error) {
	key, err := s.lookupId(url)
	switch err {
	case ErrMappingNotFound:
		key = store.GetKey(url)
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

func (s sqliteStore) lookupId(parsedUrl *nurl.URL) (uint32, error) {
	key := store.GetKey(parsedUrl)
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

	rows, err := stmt.QueryContext(s.ctx, key)
	if err != nil {
		return 0, err
	}
	defer rows.Close()
	if !rows.Next() {
		return 0, ErrMappingNotFound
	}
	var lookupId uint32
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
	if _, err := db.ExecContext(s.ctx, createSQL); err != nil {
		return err
	}
	return nil
}

func (s *sqliteStore) Delete(url *nurl.URL) (bool, error) {
	ustr := url.String()
	key := store.GetKey(ustr)
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
