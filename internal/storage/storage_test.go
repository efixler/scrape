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

func TestStore(t *testing.T) {
	s := db(t)
	defer s.Close()
	var meta resource.WebPage
	err := json.Unmarshal([]byte(mdata), &meta)
	if err != nil {
		t.Errorf("Error unmarshaling metadata: %v", err)
	}
	url, err := nurl.Parse("https://martinfowler.com/aboutMe.html#foo")
	if err != nil {
		t.Errorf("Error parsing url: %v", err)
	}
	meta.RequestedURL = url
	cText := meta.ContentText
	t.Logf("ContentText: %q", cText)
	stored := meta // this is a copy
	_, err = s.Save(&stored)
	if err != nil {
		t.Fatalf("Error storing data: %v", err)
	}
	if stored.ContentText != cText {
		t.Errorf("ContentText changed from %q to %q", cText, stored.ContentText)
	}
	//storedUrl := meta.URL()
	fetched, err := s.Fetch(url)
	// fetched, err := s.Fetch(storedUrl)
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
