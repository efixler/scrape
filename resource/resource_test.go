package resource

import (
	"encoding/json"
	"errors"
	nurl "net/url"
	"slices"
	"testing"
	"time"

	"github.com/markusmobius/go-trafilatura"
)

func TestAssertTimes(t *testing.T) {
	r := WebPage{Metadata: trafilatura.Metadata{}, ContentText: ""}
	oldNowf := nowf
	defer func() { nowf = oldNowf }()
	rightNow := time.Now()
	nowf = func() time.Time { return rightNow }

	type data struct {
		Name          string
		FetchTime     *time.Time
		TTL           *time.Duration
		wantFetchTime time.Time
		wantTTL       time.Duration
	}
	tests := []data{
		{"nils -> defaults", nil, nil, nowf(), DefaultTTL},
		{
			"Zeroes",
			&time.Time{},
			func() *time.Duration { var d time.Duration; return &d }(),
			nowf(),
			0,
		},
	}
	for _, test := range tests {
		r.FetchTime = test.FetchTime
		r.TTL = test.TTL
		r.AssertTimes()
		if r.FetchTime.IsZero() {
			t.Errorf("%s FetchTime was zero", test.Name)
		}
		if r.FetchTime.Unix() != test.wantFetchTime.Unix() {
			t.Errorf("%s FetchTime was %v, want %v", test.Name, r.FetchTime, test.wantFetchTime)
		}
		if r.TTL == nil {
			t.Errorf("%s TTL was nil", test.Name)
		}
		if *r.TTL != test.wantTTL {
			t.Errorf("%s TTL was %v, want %v", test.Name, *r.TTL, test.wantTTL)
		}
	}
}

func TestRoundTrip(t *testing.T) {
	originalURL := "http://www.example.com/index.html?utm_source=example"
	parsedOriginalURL, _ := nurl.Parse(originalURL)
	requestedURL := CleanURL(parsedOriginalURL)
	canonicalURL := "http://www.example.com/slugged"
	fetchTime := time.Now().UTC()
	r := WebPage{
		Metadata: trafilatura.Metadata{
			URL:         canonicalURL,
			Title:       "Example Title",
			Description: "An example description",
			Hostname:    "www.example.com",
			Date:        time.Date(2024, 2, 7, 0, 0, 0, 0, time.UTC),
			Sitename:    "Example Sitename",
			Image:       "http://www.example.com/image.jpg",
			Author:      "Example Author",
			Categories:  []string{"example", "test", "category"},
			Tags:        []string{"example", "test", "tag"},
			Language:    "en",
			PageType:    "article",
			ID:          "example-id",
			Fingerprint: "example-fingerprint",
			License:     "example-license",
		},
		OriginalURL:  originalURL,
		RequestedURL: requestedURL,
		FetchTime:    &fetchTime,
		StatusCode:   200,
		Error:        errors.New("example error"),
		ContentText:  "Example content",
	}
	r.AssertTimes()
	b, err := json.Marshal(r)
	if err != nil {
		t.Errorf("Error marshaling resource: %v", err)
	}
	var r2 WebPage
	err = json.Unmarshal(b, &r2)
	if err != nil {
		t.Errorf("Error unmarshaling resource: %v", err)
	}
	if r2.OriginalURL != r.OriginalURL {
		t.Errorf("Expected original URL %s, got %s", r.OriginalURL, r2.OriginalURL)
	}
	if r2.RequestedURL.String() != r.RequestedURL.String() {
		t.Errorf("Expected requested URL %s, got %s", r.RequestedURL, r2.RequestedURL)
	}
	if r2.URL().String() != r.URL().String() {
		t.Errorf("Expected URL %s, got %s", r.URL().String(), r2.URL().String())
	}
	if r2.Hostname != r.Hostname {
		t.Errorf("Expected Hostname %s, got %s", r.Hostname, r2.Hostname)
	}
	if !r2.FetchTime.Equal(*r.FetchTime) {
		t.Errorf("Expected fetch time %s, got %s", r.FetchTime, r2.FetchTime)
	}
	if r2.StatusCode != r.StatusCode {
		t.Errorf("Expected status code %d, got %d", r.StatusCode, r2.StatusCode)
	}
	if r2.Error.Error() != r.Error.Error() {
		t.Errorf("Expected error %v, got %v", r.Error, r2.Error)
	}
	if r2.Title != r.Title {
		t.Errorf("Expected title %s, got %s", r.Title, r2.Title)
	}
	if r2.Description != r.Description {
		t.Errorf("Expected description %s, got %s", r.Description, r2.Description)
	}
	if r2.Author != r.Author {
		t.Errorf("Expected author %s, got %s", r.Author, r2.Author)
	}
	if r2.Sitename != r.Sitename {
		t.Errorf("Expected sitename %s, got %s", r.Sitename, r2.Sitename)
	}
	if !r2.Date.Equal(r.Date) {
		t.Errorf("Expected fetch time %s, got %s", r.FetchTime, r2.FetchTime)
	}
	if !slices.Equal(r.Categories, r2.Categories) {
		t.Errorf("Expected categories %v, got %v", r.Categories, r2.Categories)
	}
	if !slices.Equal(r.Tags, r2.Tags) {
		t.Errorf("Expected tags %v, got %v", r.Tags, r2.Tags)
	}
	if r2.Language != r.Language {
		t.Errorf("Expected language %s, got %s", r.Language, r2.Language)
	}
	if r2.Image != r.Image {
		t.Errorf("Expected image %s, got %s", r.Image, r2.Image)
	}
	if r2.PageType != r.PageType {
		t.Errorf("Expected page type %s, got %s", r.PageType, r2.PageType)
	}
	if r2.ID != r.ID {
		t.Errorf("Expected ID %s, got %s", r.ID, r2.ID)
	}
	if r2.Fingerprint != r.Fingerprint {
		t.Errorf("Expected fingerprint %s, got %s", r.Fingerprint, r2.Fingerprint)
	}
	if r2.ContentText != r.ContentText {
		t.Errorf("Expected content text %s, got %s", r.ContentText, r2.ContentText)
	}
}

func TestEmptyDateNotSerialized(t *testing.T) {
	r := WebPage{Metadata: trafilatura.Metadata{}}
	b, err := json.Marshal(r)
	if err != nil {
		t.Errorf("Error marshaling resource: %v", err)
	}
	t.Log(string(b))
	rmap := make(map[string]interface{})
	err = json.Unmarshal(b, &rmap)
	if err != nil {
		t.Errorf("Error unmarshaling resource: %v", err)
	}
	_, exists := rmap["Date"]
	if exists {
		t.Errorf("Expected empty date to be omitted, got %v", rmap["Date"])
	}
}
