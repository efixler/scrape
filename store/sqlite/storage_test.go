package sqlite

import (
	"context"
	"encoding/json"
	nurl "net/url"
	"slices"
	"testing"

	"github.com/efixler/scrape/resource"
	"github.com/efixler/scrape/store"
)

var dbFactory store.Factory = Factory(InMemoryDBName)

func TestOpen(t *testing.T) {
	db, err := dbFactory()
	if err != nil {
		t.Errorf("Error opening database factory: %v", err)
	}
	err = db.Open(context.TODO())
	if err != nil {
		t.Fatalf("Error opening database: %v", err)
	}
	realStore, ok := db.(*SqliteStore)
	// dsn := realStore.dsn
	if !ok {
		t.Errorf("Database not of type SqliteStore")
	}
	// defer db.Close()
	err = realStore.Ping()
	if err != nil {
		t.Errorf("Error pinging database: %v", err)
	}
	err = db.Close()
	if err != nil {
		t.Errorf("Error closing database: %v", err)
	}
}

var mdata = `{
	"Title": "About Martin Fowler",
	"Author": "",
	"URL": "https://martinfowler.com/aboutMe.html",
	"Hostname": "martinfowler.com",
	"Description": "Background to Martin Fowler and martinfowler.com",
	"Sitename": "martinfowler.com",
	"Date": "1999-01-01T00:00:00Z",
	"Categories": null,
	"Tags": null,
	"ID": "",
	"Fingerprint": "",
	"License": "",
	"Language": "en",
	"Image": "https://martinfowler.com/logo-sq.png",
	"PageType": "article",
	"ContentText": "Martin Fowler"
  }`

func TestStore(t *testing.T) {
	s, err := Factory(InMemoryDBName)()
	if err != nil {
		t.Errorf("Error opening database factory: %v", err)
	}
	err = s.Open(context.TODO())
	if err != nil {
		t.Errorf("Error opening database: %v", err)
	}
	defer s.Close()
	var meta resource.WebPage
	err = json.Unmarshal([]byte(mdata), &meta)
	if err != nil {
		t.Errorf("Error unmarshaling metadata: %v", err)
	}
	url, err := nurl.Parse("https://martinfowler.com/aboutMe.html#foo")
	if err != nil {
		t.Errorf("Error parsing url: %v", err)
	}
	meta.RequestedURL = url
	cText := meta.ContentText
	stored := store.StoredUrlData{
		Data: meta,
	}
	_, err = s.Store(&stored)
	if err != nil {
		t.Errorf("Error storing data: %v", err)
	}
	if stored.Data.ContentText != cText {
		t.Errorf("ContentText changed from %q to %q", cText, stored.Data.ContentText)
	}
	//storedUrl := meta.URL()
	fetched, err := s.Fetch(url)
	// fetched, err := s.Fetch(storedUrl)
	if err != nil {
		t.Errorf("Error fetching data: %v", err)
	}
	if stored.TTL.Seconds() != fetched.TTL.Seconds() {
		t.Errorf("TTL changed from %v to %v", stored.TTL, fetched.TTL)
	}
	if stored.Data.FetchTime.Unix() != fetched.Data.FetchTime.Unix() {
		t.Errorf("FetchTime changed from %v to %v", stored.Data.FetchTime, fetched.Data.FetchTime)
	}
	if stored.Data.ContentText != fetched.Data.ContentText {
		t.Errorf("ContentText changed from %q to %q", stored.Data.ContentText, fetched.Data.ContentText)
	}
	if stored.Data.URL().String() != fetched.Data.URL().String() {
		t.Errorf("Url changed from %q to %q", stored.Data.URL(), fetched.Data.URL())
	}
	if stored.Data.RequestedURL.String() != fetched.Data.RequestedURL.String() {
		t.Errorf("Url changed from %q to %q", stored.Data.RequestedURL.String(), fetched.Data.RequestedURL.String())
	}
	if stored.Data.Title != fetched.Data.Title {
		t.Errorf("Title changed from %q to %q", stored.Data.Title, fetched.Data.Title)
	}
	if stored.Data.Author != fetched.Data.Author {
		t.Errorf("Author changed from %q to %q", stored.Data.Author, fetched.Data.Author)
	}
	if stored.Data.Hostname != fetched.Data.Hostname {
		t.Errorf("Hostname changed from %q to %q", stored.Data.Hostname, fetched.Data.Hostname)
	}
	if stored.Data.Description != fetched.Data.Description {
		t.Errorf("Description changed from %q to %q", stored.Data.Description, fetched.Data.Description)
	}
	if stored.Data.Sitename != fetched.Data.Sitename {
		t.Errorf("Sitename changed from %q to %q", stored.Data.Sitename, fetched.Data.Sitename)
	}
	if stored.Data.Date != fetched.Data.Date {
		t.Errorf("Date changed from %q to %q", stored.Data.Date, fetched.Data.Date)
	}
	if !slices.Equal(stored.Data.Categories, fetched.Data.Categories) {
		t.Errorf("Categories changed from %q to %q", stored.Data.Categories, fetched.Data.Categories)
	}
	if !slices.Equal(stored.Data.Tags, fetched.Data.Tags) {
		t.Errorf("Tags changed from %q to %q", stored.Data.Tags, fetched.Data.Tags)
	}
	if stored.Data.ID != fetched.Data.ID {
		t.Errorf("ID changed from %q to %q", stored.Data.ID, fetched.Data.ID)
	}
	if stored.Data.Fingerprint != fetched.Data.Fingerprint {
		t.Errorf("Fingerprint changed from %q to %q", stored.Data.Fingerprint, fetched.Data.Fingerprint)
	}
	if stored.Data.License != fetched.Data.License {
		t.Errorf("License changed from %q to %q", stored.Data.License, fetched.Data.License)
	}
	if stored.Data.Language != fetched.Data.Language {
		t.Errorf("Language changed from %q to %q", stored.Data.Language, fetched.Data.Language)
	}
	if stored.Data.Image != fetched.Data.Image {
		t.Errorf("Image changed from %q to %q", stored.Data.Image, fetched.Data.Image)
	}
	if stored.Data.PageType != fetched.Data.PageType {
		t.Errorf("PageType changed from %q to %q", stored.Data.PageType, fetched.Data.PageType)
	}
	// NB: Delete only works for canonical URLs
	rs, _ := s.(*SqliteStore)
	ok, err := rs.delete(url)
	if err != nil {
		t.Errorf("Unexpected error deleting non-canonical record: %v", err)
	}
	if ok {
		t.Errorf("Delete returned true, deleted non-canonical record (url: %s)", url)
	}

	ok, err = rs.delete(stored.Data.URL())
	if err != nil {
		t.Errorf("Error deleting record: %v", err)
	} else if !ok {
		t.Errorf("Delete returned false, didn't delete record (url: %s)", url)
	}

}
