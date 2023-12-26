package sqlite

import (
	"context"
	"database/sql"
	_ "embed"
	"encoding/json"
	"errors"
	"fmt"
	nurl "net/url"
	"time"

	"log/slog"

	"github.com/efixler/scrape/resource"
	"github.com/efixler/scrape/store"
	"github.com/efixler/scrape/store/internal/database"

	_ "github.com/mattn/go-sqlite3"
)

const (
	DEFAULT_BUSY_TIMEOUT = 5 * time.Second
	DEFAULT_JOURNAL_MODE = "WAL"
	DEFAULT_CACHE_SIZE   = 20000
	SMALL_CACHE_SIZE     = 2000 // This is actually the sqlite default
	SQLITE_SYNC_OFF      = "OFF"
	SQLITE_SYNC_NORMAL   = "NORMAL"
	DEFAULT_SYNC         = SQLITE_SYNC_OFF
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
	ErrMappingNotFound = errors.New("id mapping not found")
	ErrStoreNotOpen    = errors.New("store not opened for this dsn")
	ErrNoDatabase      = errors.New("the database did not exist")
)

// Returns the factory function that will be used to instantiate the store.
// The factory function will guarantee that the preconditions are in place for
// the db and the instance is ready to use.
func Factory(filename string) store.Factory {
	options := DefaultOptions()
	dsnF := func() string {
		return dsn(filename, options)
	}
	// If the factory function returns successfully, then we have a valid DSN
	// and we've made any local directories needed to support it.
	return func() (store.URLDataStore, error) {
		s := &sqliteStore{
			DBHandle: database.DBHandle[stmtIndex]{
				Driver: database.SQLite,
				DSN:    dsnF,
			},
			filename: filename,
			options:  DefaultOptions(),
		}
		var err error
		s.resolvedPath, err = dbPath(filename)
		if err != nil {
			switch err {
			case ErrIsInMemory:
				options.createIfNotExists = true // always create an in-memory DB
				// continue below if the caller wants an in-memory DB
			default:
				// if we couldn't resolve the path, we won't be able to open or create
				return nil, err
			}
		}
		if (err == nil) && !exists(s.resolvedPath) {
			if err = s.createPathToDB(); err != nil {
				return nil, errors.Join(ErrNoDatabase, err)
			}
		}
		return s, nil
	}
}

type sqliteStore struct {
	database.DBHandle[stmtIndex]
	filename     string
	resolvedPath string
	options      SqliteOptions
}

func (s *sqliteStore) Open(ctx context.Context) error {
	if s.DB != nil {
		return database.ErrDatabaseAlreadyOpen
	}
	err := s.DBHandle.Open(ctx)
	if err != nil {
		return err
	}
	// SQLite will open even if the the DB file is not present, it will only fail later.
	// So, if the db hasn't been opened, check for the file here.
	// In Memory DBs must always be created
	if (s.filename == InMemoryDBName) || !exists(s.resolvedPath) {
		if err := s.create(); err != nil {
			return err
		}
	}
	return nil
}

// The underlying DB handle's close will be called when the context
// passed to Open() is cancelled
func (s *sqliteStore) Close() error {
	err := s.DBHandle.Close()
	if err != nil {
		slog.Warn("error closing sqlite store", "dsn", s.DSN(), "error", err)
	}
	return err
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
	stmt, err := s.Statement(Store, func(ctx context.Context, db *sql.DB) (*sql.Stmt, error) {
		return db.PrepareContext(ctx, qStore)
	})
	if err != nil {
		return 0, err
	}
	result, err := stmt.ExecContext(s.Ctx, values...)
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
	stmt, err := s.Statement(StoreId, func(ctx context.Context, db *sql.DB) (*sql.Stmt, error) {
		return db.PrepareContext(ctx, qStoreId)
	})
	if err != nil {
		return err
	}
	_, err = stmt.ExecContext(s.Ctx, store.Key(parsedUrl), canonicalId)
	if err != nil {
		return err
	}
	return nil
}

// Fetch will return the stored data for requested URL, or nil if not found.
// If the requested URL matches a canonical URL AND the requested URL has not been fetched
// before, then we'll return the previously data for the canonical URL.
// We don't get canonical on the first try, since the canonical is derived from the page's parse,
// and it hasn't been parsed yet here. This has the side effect of letting the caller add arbitrary
// parameters to force a page re-fetch.
func (s sqliteStore) Fetch(url *nurl.URL) (*store.StoredUrlData, error) {
	requested_key := store.Key(url)
	key, err := s.lookupId(requested_key)
	switch err {
	case ErrMappingNotFound:
		slog.Debug("sqlite: No mapped key for resource, trying direct key", "url", url.String(), "requested_key", requested_key, "canonical_key", key)
		key = requested_key
	case nil:
		// we have a key
		slog.Debug("sqlite: Found mapped key for resource", "url", url.String(), "requested_key", requested_key, "canonical_key", key)
	default:
		return nil, err
	}
	stmt, err := s.Statement(Fetch, func(ctx context.Context, db *sql.DB) (*sql.Stmt, error) {
		return db.PrepareContext(ctx, qFetch)
	})
	if err != nil {
		return nil, err
	}
	rows, err := stmt.QueryContext(s.Ctx, key)
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
	stmt, err := s.Statement(LookupId, func(ctx context.Context, db *sql.DB) (*sql.Stmt, error) {
		return db.PrepareContext(ctx, qLookupId)
	})
	if err != nil {
		return 0, err
	}
	rows, err := stmt.QueryContext(s.Ctx, requested_id)
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
	// TODO: This probably shoold be the same as CreateDB (espcially now that that query flushes the DB)
	if s.DB == nil {
		return ErrStoreNotOpen
	}
	if _, err := s.DB.ExecContext(s.Ctx, qClear); err != nil {
		return err
	}
	return nil
}

// Delete will only delete a url that matches the canonical URL.
// TODO: Evaluate desired behavior here
func (s *sqliteStore) Delete(url *nurl.URL) (bool, error) {
	key := store.Key(url)
	stmt, err := s.Statement(Delete, func(ctx context.Context, db *sql.DB) (*sql.Stmt, error) {
		return db.PrepareContext(ctx, qDelete)
	})
	if err != nil {
		return false, err
	}
	result, err := stmt.ExecContext(s.Ctx, key)
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
