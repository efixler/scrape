/*
This is the implementation of the store.URLDataStore interface for sqlite.

Use New() to make a new sqlite storage instance.
  - You *must* call Open()
  - The DB will be closed when the context passed to Open() is cancelled.
  - Concurrent usage OK
  - In-Memory DBs are supported
  - The DB will be created if it doesn't exist
*/
package sqlite

import (
	"context"
	_ "embed"
	"errors"
	"time"

	"github.com/efixler/scrape/database"
	"github.com/efixler/scrape/internal/storage"
	"github.com/efixler/scrape/store"

	_ "github.com/mattn/go-sqlite3"
)

// const (
// 	qStore    = `REPLACE INTO urls (id, url, parsed_url, fetch_time, expires, metadata, content_text) VALUES (?, ?, ?, ?, ?, ?, ?)`
// 	qClear    = `DELETE FROM urls; DELETE FROM id_map`
// 	qLookupId = `SELECT canonical_id FROM id_map WHERE requested_id = ?`
// 	qStoreId  = `REPLACE INTO id_map (requested_id, canonical_id) VALUES (?, ?)`
// 	qClearId  = `DELETE FROM id_map where canonical_id = ?`
// 	qFetch    = `SELECT url, parsed_url, fetch_time, expires, metadata, content_text FROM urls WHERE id = ?`
// 	qDelete   = `DELETE FROM urls WHERE id = ?`
// )

// type stmtIndex int

// const (
// 	_ stmtIndex = iota
// 	save
// 	clear
// 	lookupId
// 	saveId
// 	fetch
// 	clearId
// 	delete
// )

var (
	ErrStoreNotOpen       = errors.New("store not opened for this dsn")
	ErrCantCreateDatabase = errors.New("can't create the database")
)

// Returns the factory function that can be used to instantiate a sqlite store
// in the cases where either creation should be delayed or where the caller may
// want to instantiate multiple stores with the same configuration.
func Factory(options ...option) store.Factory {
	return func() (store.URLDataStore, error) {
		return New(options...)
	}
}

func New(options ...option) (store.URLDataStore, error) {
	s := &Store{
		SQLStorage: storage.New(database.SQLite),
	}
	c := &config{}
	Defaults()(c)
	for _, opt := range options {
		if err := opt(c); err != nil {
			return nil, err
		}
	}
	s.config = *c
	s.DBHandle.DSNSource = s.config
	return s, nil
}

type Store struct {
	*storage.SQLStorage
	config config
	stats  *Stats
}

// Opens the database, creating it if it doesn't exist.
// The passed contexts will be used for query preparation, and to
// close the database when the context is cancelled.
func (s *Store) Open(ctx context.Context) error {
	err := s.DBHandle.Open(ctx)
	if err != nil {
		return err
	}
	// SQLite will open even if the the DB file is not present, it will only fail later.
	// So, if the db hasn't been opened, check for the file here.
	// In Memory DBs must always be created
	inMemory := s.config.IsInMemory()
	needsCreate := inMemory || !exists(s.config.filename)
	if needsCreate {
		if err := s.Create(); err != nil {
			return err
		}
	}
	if inMemory {
		// Unfortunately, SQLite in-memory DBs are bound to a single connection.
		s.DB.SetMaxOpenConns(1)
		s.DB.SetMaxIdleConns(1)
		s.DB.SetConnMaxLifetime(-1)
	}
	s.Maintenance(24*time.Hour, maintain)
	return nil
}

// Save the data for a URL. Will overwrite data where the URL is the same.
// Returns a key for the stored URL (which you actually can't
// use for anything, so this interface may change)
// func (s *Store) Save(uptr *resource.WebPage) (uint64, error) {
// 	uptr.AssertTimes()           // modify the original with times if needed
// 	key := store.Key(uptr.URL()) // key is for the canonical URL
// 	metadata, err := store.SerializeMetadata(uptr)
// 	if err != nil {
// 		return 0, err
// 	}

// 	// (id, url, parsed_url, fetch_time, expires, metadata, content_text)
// 	values := []any{
// 		key,
// 		uptr.URL().String(),
// 		uptr.RequestURL().String(),
// 		uptr.FetchTime.Unix(),
// 		uptr.ExpireTime().Unix(),
// 		string(metadata),
// 		uptr.ContentText,
// 	}
// 	stmt, err := s.Statement(save, func(ctx context.Context, db *sql.DB) (*sql.Stmt, error) {
// 		return db.PrepareContext(ctx, qStore)
// 	})
// 	if err != nil {
// 		return 0, err
// 	}
// 	result, err := stmt.ExecContext(s.Ctx, values...)
// 	if err != nil {
// 		return 0, err
// 	}
// 	// todo: this can fail silently
// 	// todo: test case for this, including self-mapping when the canonical url is the same as the requested url
// 	err_id_map := s.storeIdMap(uptr.RequestURL(), key)

// 	rows, err := result.RowsAffected()
// 	if err != nil {
// 		return 0, errors.Join(err, err_id_map)
// 	}
// 	if rows != 1 {
// 		return 0, errors.Join(fmt.Errorf("expected 1 row affected, got %d", rows), err_id_map)
// 	}
// 	return key, nil
// }

// func (s Store) storeIdMap(parsedUrl *nurl.URL, canonicalId uint64) error {
// 	stmt, err := s.Statement(saveId, func(ctx context.Context, db *sql.DB) (*sql.Stmt, error) {
// 		return db.PrepareContext(ctx, qStoreId)
// 	})
// 	if err != nil {
// 		return err
// 	}
// 	_, err = stmt.ExecContext(s.Ctx, store.Key(parsedUrl), canonicalId)
// 	if err != nil {
// 		return err
// 	}
// 	return nil
// }

// Fetch will return the stored data for requested URL, or nil if not found.
//
// The returned result _may_ come from a different URL than the requested URL, if
// we've seen the passed URL before AND the page reported it's canonical url as
// being different than the requested URL.
//
// In that case, the canonical version of the content will be returned, if we have it.
// func (s Store) Fetch(url *nurl.URL) (*resource.WebPage, error) {
// 	requested_key := store.Key(url)
// 	key, err := s.lookupId(requested_key)
// 	switch err {
// 	case store.ErrMappingNotFound:
// 		slog.Debug("sqlite: No mapped key for resource, trying direct key", "url", url.String(), "requested_key", requested_key, "canonical_key", key)
// 		key = requested_key
// 	case nil:
// 		// we have a key
// 		slog.Debug("sqlite: Found mapped key for resource", "url", url.String(), "requested_key", requested_key, "canonical_key", key)
// 	default:
// 		return nil, err
// 	}
// 	stmt, err := s.Statement(fetch, func(ctx context.Context, db *sql.DB) (*sql.Stmt, error) {
// 		return db.PrepareContext(ctx, qFetch)
// 	})
// 	if err != nil {
// 		return nil, err
// 	}
// 	rows, err := stmt.QueryContext(s.Ctx, key)
// 	if err != nil {
// 		return nil, err
// 	}
// 	defer rows.Close()
// 	if !rows.Next() {
// 		return nil, store.ErrorResourceNotFound
// 	}
// 	// parsed_url, fetch_time, expires, metadata, content_text
// 	var (
// 		canonicalUrl string
// 		parsedUrl    string
// 		fetchEpoch   int64
// 		expiryEpoch  int64
// 		metadata     string
// 		contentText  string
// 	)
// 	err = rows.Scan(&canonicalUrl, &parsedUrl, &fetchEpoch, &expiryEpoch, &metadata, &contentText)
// 	if err != nil {
// 		return nil, err
// 	}
// 	exptime := time.Unix(expiryEpoch, 0)
// 	if time.Now().After(exptime) {
// 		return nil, store.ErrorResourceNotFound
// 	}
// 	fetchTime := time.Unix(fetchEpoch, 0).UTC()
// 	ttl := exptime.Sub(fetchTime)
// 	page := &resource.WebPage{FetchTime: &fetchTime}
// 	page.Metadata.URL = canonicalUrl
// 	page.RequestedURL, err = nurl.Parse(parsedUrl)
// 	if err != nil {
// 		return nil, err
// 	}
// 	err = json.Unmarshal([]byte(metadata), page)
// 	if err != nil {
// 		return nil, err
// 	}
// 	page.ContentText = contentText
// 	page.TTL = &ttl

// 	//fmt.Println(parsedUrl, fetchEpoch, expiryEpoch, metadata, contentText)
// 	return page, nil
// }

// Will search url_ids to see if there's a parent entry for this url.
// func (s Store) lookupId(requested_id uint64) (uint64, error) {
// 	stmt, err := s.Statement(lookupId, func(ctx context.Context, db *sql.DB) (*sql.Stmt, error) {
// 		return db.PrepareContext(ctx, qLookupId)
// 	})
// 	if err != nil {
// 		return 0, err
// 	}
// 	rows, err := stmt.QueryContext(s.Ctx, requested_id)
// 	if err != nil {
// 		return 0, err
// 	}
// 	defer rows.Close()
// 	if !rows.Next() {
// 		return 0, store.ErrMappingNotFound
// 	}
// 	var lookupId uint64
// 	err = rows.Scan(&lookupId)
// 	if err != nil {
// 		return 0, err
// 	}
// 	return lookupId, nil
// }

// Delete will only delete a url that matches the canonical URL.
// TODO: Evaluate desired behavior here
// func (s *Store) Delete(url *nurl.URL) (bool, error) {
// 	key := store.Key(url)
// 	stmt, err := s.Statement(delete, func(ctx context.Context, db *sql.DB) (*sql.Stmt, error) {
// 		return db.PrepareContext(ctx, qDelete)
// 	})
// 	if err != nil {
// 		return false, err
// 	}
// 	result, err := stmt.ExecContext(s.Ctx, key)
// 	if err != nil {
// 		return false, err
// 	}
// 	rows, err := result.RowsAffected()
// 	if err != nil {
// 		return false, err
// 	}
// 	switch rows {
// 	case 0:
// 		return false, nil
// 	case 1:
// 		return true, nil
// 	default:
// 		return false, fmt.Errorf("expected 0 or 1 row affected, got %d", rows)
// 	}
// }
