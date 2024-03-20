package storage

import (
	"encoding/json"
	nurl "net/url"
	"slices"
	"testing"
	"time"

	"github.com/efixler/scrape/resource"
	"github.com/efixler/scrape/store"
)

type dsnGen string

var dsn = dsnGen(dbURL)

func (d dsnGen) String() string {
	return string(d)
}

func (d dsnGen) DSN() string {
	return string(d)
}

func TestOpen(t *testing.T) {
	db := db(t)
	err := db.Ping()
	if err != nil {
		t.Errorf("Error pinging database: %v", err)
	}
	err = db.Close()
	if err != nil {
		t.Errorf("Error closing database: %v", err)
	}
}

var mdata = `{
	"title": "About Martin Fowler",
	"author": "Martin Fowler",
	"url": "https://martinfowler.com/aboutMe.html",
	"requested_url": "https://martinfowler.com/aboutMe.html?bar=baz",
	"original_url": "https://martinfowler.com/aboutMe.html#foo?bar=baz",
	"hostname": "martinfowler.com",
	"description": "Background to Martin Fowler and martinfowler.com",
	"sitename": "martinfowler.com",
	"date": "1999-01-01T00:00:00Z",
	"categories": null,
	"tags": null,
	"id": "",
	"fingerprint": "",
	"license": "",
	"language": "en",
	"image": "https://martinfowler.com/logo-sq.png",
	"page_type": "article",
	"content_text": "Martin Fowler"
  }`

// TODO: Fuzz this so every return is different
func getWebPage(t *testing.T) *resource.WebPage {
	var meta resource.WebPage
	err := json.Unmarshal([]byte(mdata), &meta)
	if err != nil {
		t.Errorf("Error unmarshaling metadata: %v", err)
	}
	return &meta
}

func TestStore(t *testing.T) {
	s := db(t)
	defer s.Close()
	meta := getWebPage(t)
	cText := meta.ContentText

	stored := *meta // this is a copy
	url := meta.RequestedURL
	canonicalId, err := s.Save(&stored)
	if err != nil {
		t.Fatalf("Error storing data: %v", err)
	}
	if stored.ContentText != cText {
		t.Errorf("ContentText changed from %q to %q", cText, stored.ContentText)
	}
	fetched, err := s.Fetch(url)
	if err != nil {
		t.Fatalf("Error fetching data: %v", err)
	}
	if stored.TTL.Seconds() != fetched.TTL.Seconds() {
		t.Errorf("TTL changed from %v to %v", stored.TTL, fetched.TTL)
	}
	if stored.FetchTime.Unix() != fetched.FetchTime.Unix() {
		t.Errorf("FetchTime changed from %v to %v", stored.FetchTime, fetched.FetchTime)
	}
	if stored.ContentText != fetched.ContentText {
		t.Errorf("ContentText changed from %q to %q", stored.ContentText, fetched.ContentText)
	}
	if stored.URL().String() != fetched.URL().String() {
		t.Errorf("Url changed from %q to %q", stored.URL(), fetched.URL())
	}
	if stored.RequestedURL.String() != fetched.RequestedURL.String() {
		t.Errorf("Url changed from %q to %q", stored.RequestedURL.String(), fetched.RequestedURL.String())
	}
	if fetched.OriginalURL != "" {
		t.Errorf("OriginalURL should be empty, got %q", stored.OriginalURL)
	}
	if stored.Title != fetched.Title {
		t.Errorf("Title changed from %q to %q", stored.Title, fetched.Title)
	}
	if stored.Author != fetched.Author {
		t.Errorf("Author changed from %q to %q", stored.Author, fetched.Author)
	}
	if stored.Hostname != fetched.Hostname {
		t.Errorf("Hostname changed from %q to %q", stored.Hostname, fetched.Hostname)
	}
	if stored.Description != fetched.Description {
		t.Errorf("Description changed from %q to %q", stored.Description, fetched.Description)
	}
	if stored.Sitename != fetched.Sitename {
		t.Errorf("Sitename changed from %q to %q", stored.Sitename, fetched.Sitename)
	}
	if stored.Date != fetched.Date {
		t.Errorf("Date changed from %q to %q", stored.Date, fetched.Date)
	}
	if !slices.Equal(stored.Categories, fetched.Categories) {
		t.Errorf("Categories changed from %q to %q", stored.Categories, fetched.Categories)
	}
	if !slices.Equal(stored.Tags, fetched.Tags) {
		t.Errorf("Tags changed from %q to %q", stored.Tags, fetched.Tags)
	}
	if stored.ID != fetched.ID {
		t.Errorf("ID changed from %q to %q", stored.ID, fetched.ID)
	}
	if stored.Fingerprint != fetched.Fingerprint {
		t.Errorf("Fingerprint changed from %q to %q", stored.Fingerprint, fetched.Fingerprint)
	}
	if stored.License != fetched.License {
		t.Errorf("License changed from %q to %q", stored.License, fetched.License)
	}
	if stored.Language != fetched.Language {
		t.Errorf("Language changed from %q to %q", stored.Language, fetched.Language)
	}
	if stored.Image != fetched.Image {
		t.Errorf("Image changed from %q to %q", stored.Image, fetched.Image)
	}
	if stored.PageType != fetched.PageType {
		t.Errorf("PageType changed from %q to %q", stored.PageType, fetched.PageType)
	}

	// check that the expected lookup between requested and canonical URLs is correct
	if lid, err := s.lookupId(Key(url)); lid != canonicalId {
		t.Errorf("Expected lookup id %d, got %d (err: %s)", canonicalId, lid, err)
	}

	// NB: Delete only works for canonical URLs
	ok, err := s.Delete(url)
	if err != nil {
		t.Errorf("Unexpected error deleting non-canonical record: %v", err)
	}
	if ok {
		t.Errorf("Delete returned true, deleted non-canonical record (url: %s)", url)
	}
	ok, err = s.Delete(stored.URL())
	if err != nil {
		t.Errorf("Error deleting record: %v", err)
	} else if !ok {
		t.Errorf("Delete returned false, didn't delete record (url: %s)", url)
	}
}

func TestReturnValuesWhenResourceNotExists(t *testing.T) {
	s := db(t)
	defer s.Close()
	url, err := nurl.Parse("https://martinfowler.com/aboutYou")
	if err != nil {
		t.Errorf("Error parsing url: %v", err)
	}
	res, err := s.Fetch(url)
	if err != store.ErrorResourceNotFound {
		t.Errorf("Expected error %v, got %v", store.ErrorResourceNotFound, err)
	}
	if res != nil {
		t.Errorf("Expected nil resource, got %v", res)
	}
}

func TestReturnValuesWhenResourceIsExpired(t *testing.T) {
	s := db(t)
	defer s.Close()
	var meta resource.WebPage
	err := json.Unmarshal([]byte(mdata), &meta)
	if err != nil {
		t.Errorf("Error unmarshaling metadata: %v", err)
	}
	url, err := nurl.Parse("https://martinfowler.com/aboutThem")
	if err != nil {
		t.Errorf("Error parsing url: %v", err)
	}
	meta.RequestedURL = url
	ttl := time.Duration(0)
	meta.TTL = &ttl
	_, err = s.Save(&meta)
	if err != nil {
		t.Errorf("Error storing data: %v", err)
	}
	res, err := s.Fetch(url)
	if err != store.ErrorResourceNotFound {
		t.Errorf("Expected error %v, got %v", store.ErrorResourceNotFound, err)
	}
	if res != nil {
		t.Errorf("Expected nil resource, got %v", res)
	}
}

// We store self-referential lookups. This test confirms that they are stored.
func TestCanonicalSelfLookupExists(t *testing.T) {
	s := db(t)
	defer s.Close()
	url, _ := nurl.Parse("https://martinfowler.com/aboutMe.html")
	key := Key(url)
	err := s.storeIdMap(url, key) // stores a self-referential lookup
	if err != nil {
		t.Fatalf("Error storing id lookup: %v", err)
	}
	id, err := s.lookupId(key)
	if err != nil {
		t.Fatalf("Error looking up id: %v", err)
	}
	if id != key {
		t.Errorf("Expected id %d, got %d", key, id)
	}
}

func TestClear(t *testing.T) {
	s := db(t)
	defer s.Close()
	res := getWebPage(t)
	_, err := s.Save(res)
	if err != nil {
		t.Fatalf("Error storing data: %v", err)
	}
	err = s.Clear()
	if err != nil {
		t.Errorf("Error clearing store: %v", err)
	}
	if rows, err := s.DB.QueryContext(s.Ctx, "SELECT COUNT(*) FROM urls"); err != nil {
		t.Fatalf("Error counting rows after insert: %v", err)
	} else {
		defer rows.Close()
		rows.Next()
		var count int
		rows.Scan(&count)
		if count != 0 {
			t.Errorf("Expected no rows, got %d", count)
		}
	}
}

func TestDelete(t *testing.T) {
	s := db(t)
	defer s.Close()
	res := getWebPage(t)
	_, err := s.Save(res)
	if err != nil {
		t.Fatalf("Error storing data: %v", err)
	}
	ok, err := s.Delete(res.URL())
	if err != nil {
		t.Errorf("Error deleting record: %v", err)
	}
	if !ok {
		t.Errorf("Delete returned false, didn't delete record (url: %s)", res.URL())
	}
}
