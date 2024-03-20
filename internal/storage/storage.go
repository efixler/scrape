package storage

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"log/slog"
	nurl "net/url"
	"time"

	"github.com/efixler/scrape/database"
	"github.com/efixler/scrape/resource"
	"github.com/efixler/scrape/store"
)

type stmtIndex int

const (
	_ stmtIndex = iota
	save
	saveId
	lookupId
	fetch
	delete
)

const (
	qSave     = `REPLACE INTO urls (id, url, parsed_url, fetch_time, expires, metadata, content_text) VALUES (?, ?, ?, ?, ?, ?, ?);`
	qSaveId   = `REPLACE INTO id_map (requested_id, canonical_id) VALUES (?, ?)`
	qLookupId = `SELECT canonical_id FROM id_map WHERE requested_id = ?`
	qFetch    = `SELECT url, parsed_url, fetch_time, expires, metadata, content_text FROM urls WHERE id = ?`
	qDelete   = `DELETE FROM urls WHERE id = ?`
	qClear    = `DELETE FROM urls; DELETE FROM id_map;`
	// qClearId  = `DELETE FROM id_map where canonical_id = ?`
)

type SQLStorage struct {
	*database.DBHandle[stmtIndex]
}

func New(driver database.DriverName) *SQLStorage {
	return &SQLStorage{
		DBHandle: &database.DBHandle[stmtIndex]{
			Driver: driver,
		},
	}
}

// Save the data for a URL. Will overwrite data where the URL is the same.
// Save() will use the canonical url (.URL()) of the passed resource both for the key
// and for the url field in the stored data. It will also store an id map entry
// for the requested URL, back to the canonical URL. This mapping will also be stored in
// cases where the two urls are the same.
// Returns a key for the stored URL (which you actually can't
// use for anything, so this interface may change)
func (s *SQLStorage) Save(uptr *resource.WebPage) (uint64, error) {
	uptr.AssertTimes()
	key := store.Key(uptr.URL())
	metadata, err := store.SerializeMetadata(uptr)
	if err != nil {
		return 0, err
	}
	values := []any{
		key,
		uptr.URL().String(),
		uptr.RequestURL().String(),
		uptr.FetchTime.Unix(),
		uptr.ExpireTime().Unix(),
		string(metadata),
		uptr.ContentText,
	}

	stmt, err := s.Statement(save, func(ctx context.Context, db *sql.DB) (*sql.Stmt, error) {
		return db.PrepareContext(ctx, qSave)
	})
	if err != nil {
		return 0, err
	}
	result, err := stmt.ExecContext(s.Ctx, values...)
	if err != nil {
		return 0, err
	}
	rows, err := result.RowsAffected()
	if err != nil {
		return 0, err
	}
	if (rows == 0) || (rows > 2) {
		return 0, fmt.Errorf("expected 1 row affected, got %d", rows)
	}
	// TODO: Test case
	// TODO: Clarify intent when canonical = requested
	err = s.storeIdMap(uptr.RequestURL(), key)
	if err != nil {
		return 0, err
	}
	return key, nil
}

func (s SQLStorage) storeIdMap(requested *nurl.URL, canonicalID uint64) error {
	stmt, err := s.Statement(saveId, func(ctx context.Context, db *sql.DB) (*sql.Stmt, error) {
		return db.PrepareContext(ctx, qSaveId)
	})
	if err != nil {
		return err
	}
	_, err = stmt.ExecContext(s.Ctx, store.Key(requested), canonicalID)
	if err != nil {
		return err
	}
	return nil
}

// Fetch will return the stored data for requested URL, or nil if not found.
//
// The returned result _may_ come from a different URL than the requested URL, if
// we've seen the passed URL before AND the page reported it's canonical url as
// being different than the requested URL.
//
// In that case, the canonical version of the content will be returned, if we have it.
func (s SQLStorage) Fetch(url *nurl.URL) (*resource.WebPage, error) {
	requested_key := store.Key(url)
	key, err := s.lookupId(requested_key)
	switch err {
	case store.ErrMappingNotFound:
		slog.Debug("sqlite: No mapped key for resource, trying direct key", "url", url.String(), "requested_key", requested_key, "canonical_key", key)
		key = requested_key
	case nil:
		// we have a key
		slog.Debug("sqlite: Found mapped key for resource", "url", url.String(), "requested_key", requested_key, "canonical_key", key)
	default:
		return nil, err
	}
	stmt, err := s.Statement(fetch, func(ctx context.Context, db *sql.DB) (*sql.Stmt, error) {
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
		return nil, store.ErrorResourceNotFound
	}
	// parsed_url, fetch_time, expires, metadata, content_text
	var (
		canonicalUrl string
		parsedUrl    string
		fetchEpoch   int64
		expiryEpoch  int64
		metadata     string
		contentText  string
	)
	err = rows.Scan(&canonicalUrl, &parsedUrl, &fetchEpoch, &expiryEpoch, &metadata, &contentText)
	if err != nil {
		return nil, err
	}
	exptime := time.Unix(expiryEpoch, 0)
	if time.Now().After(exptime) {
		return nil, store.ErrorResourceNotFound
	}
	fetchTime := time.Unix(fetchEpoch, 0).UTC()
	ttl := exptime.Sub(fetchTime)
	page := &resource.WebPage{FetchTime: &fetchTime}
	page.Metadata.URL = canonicalUrl
	page.RequestedURL, err = nurl.Parse(parsedUrl)
	if err != nil {
		return nil, err
	}
	err = json.Unmarshal([]byte(metadata), page)
	if err != nil {
		return nil, err
	}
	page.ContentText = contentText
	page.TTL = &ttl

	//fmt.Println(parsedUrl, fetchEpoch, expiryEpoch, metadata, contentText)
	return page, nil
}

// Will search url_ids to see if there's a parent entry for this url.
func (s *SQLStorage) lookupId(requested_id uint64) (uint64, error) {
	stmt, err := s.Statement(lookupId, func(ctx context.Context, db *sql.DB) (*sql.Stmt, error) {
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
		return 0, store.ErrMappingNotFound
	}
	var lookupId uint64
	err = rows.Scan(&lookupId)
	if err != nil {
		return 0, err
	}
	return lookupId, nil
}

// Delete will only delete a url that matches the canonical URL.
// TODO: Evaluate desired behavior here
// TODO: Not accounting for lookup keys
// NB: TTL management is handled by maintenance routines
func (s *SQLStorage) Delete(url *nurl.URL) (bool, error) {
	key := store.Key(url)
	stmt, err := s.Statement(delete, func(ctx context.Context, db *sql.DB) (*sql.Stmt, error) {
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

// Clear will delete all content from the database
func (s *SQLStorage) Clear() error {
	_, err := s.DB.ExecContext(s.Ctx, qClear)
	return err
}
