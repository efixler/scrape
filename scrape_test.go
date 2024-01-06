package scrape

import (
	"context"
	_ "embed"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	nurl "net/url"
	"testing"
	"time"

	"github.com/efixler/scrape/fetch/trafilatura"
	"github.com/efixler/scrape/resource"
	"github.com/efixler/scrape/store/sqlite"
)

//go:embed samples/arstechnica-1993804.html
var testPage []byte

//go:embed samples/arstechnica-1993804.json
var testJson []byte

func TestFetchStoresAndRetrieves(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		w.WriteHeader(http.StatusOK)
		w.Write(testPage)
	}))
	defer ts.Close()
	client := ts.Client()
	fOptions := *trafilatura.DefaultOptions
	fOptions.HttpClient = client
	fFactory := trafilatura.Factory(fOptions)
	sFactory := sqlite.Factory(sqlite.InMemoryDBName)

	fetcher, err := NewStorageBackedFetcher(fFactory, sFactory)
	if err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	err = fetcher.Open(ctx)
	if err != nil {
		t.Fatal(err)
	}
	baseUrl := ts.URL + "/arstechnica-1993804.html?keep_param=1"
	url := baseUrl + "&utm_source=feedburner"
	netURL, _ := nurl.Parse(url)
	fetched, err := fetcher.Fetch(netURL)
	if err != nil {
		t.Errorf("Expected no error for %s, got %s", url, err)
	}
	if fetched.OriginalURL != url {
		t.Errorf("Expected URL %s for fetched resource, got %s", url, fetched.OriginalURL)
	}
	if fetched.RequestedURL.String() != baseUrl {
		t.Errorf("Expected URL %s for fetched resource, got %s", baseUrl, fetched.RequestedURL.String())
	}
	if fetched.StatusCode != http.StatusOK {
		t.Errorf("Expected status code %d for fetched resource, got %d", http.StatusOK, fetched.StatusCode)
	}
	reference := &resource.WebPage{}
	err = json.Unmarshal(testJson, reference)
	if err != nil {
		t.Fatal(err)
	}
	time.Sleep(1 * time.Second) // make sure wallclock has advanced so we can check the fetch time
	stored, err := fetcher.Fetch(netURL)
	if err != nil {
		t.Errorf("Expected no error for %s, got %s", url, err)
	}

	for i, test := range []*resource.WebPage{fetched, stored} {
		var label string
		if i == 0 {
			label = "fetched"
		} else {
			label = "stored"
		}
		if test.StatusCode != reference.StatusCode {
			t.Errorf(
				"Expected status code %d for %s resource, got %d",
				reference.StatusCode,
				label,
				test.StatusCode)
		}
		if test.Title != reference.Title {
			t.Errorf("Expected title %s for %s resource, got %s", reference.Title, label, test.Title)
		}
		if test.Description != reference.Description {
			t.Errorf(
				"Expected description %s for %s resource, got %s",
				reference.Description,
				label,
				test.Description,
			)
		}
		if test.Author != reference.Author {
			t.Errorf("Expected author %s for %s resource, got %s", reference.Author, label, test.Author)
		}
		if test.Sitename != reference.Sitename {
			t.Errorf("Expected sitename %s for %s resource, got %s", reference.Sitename, label, test.Sitename)
		}
		if test.Date != reference.Date {
			t.Errorf("Expected date %s for %s resource, got %s", reference.Date, label, test.Date)
		}
		if test.Language != reference.Language {
			t.Errorf("Expected language %s for %s resource, got %s", reference.Language, label, test.Language)
		}
		if test.Image != reference.Image {
			t.Errorf("Expected image %s for %s resource, got %s", reference.Image, label, test.Image)
		}
		if test.PageType != reference.PageType {
			t.Errorf("Expected page type %s for %s resource, got %s", reference.PageType, label, test.PageType)
		}
		if test.PageType != reference.PageType {
			t.Errorf("Expected page type %s for %s resource, got %s", reference.PageType, label, test.PageType)
		}
		if test.URL().String() != reference.URL().String() {
			t.Errorf("Expected URL %s for %s resource, got %s", reference.URL().String(), label, test.URL().String())
		}
		if test.Hostname != reference.Hostname {
			t.Errorf("Expected hostname %s for %s resource, got %s", reference.Hostname, label, test.Hostname)
		}
		if test.ContentText != reference.ContentText {
			t.Errorf("Expected content text %s for %s resource, got %s", reference.ContentText, label, test.ContentText)
		}
	}
	if !fetched.FetchTime.Equal(*stored.FetchTime) {
		t.Errorf("Expected fetch time %s for stored resource, got %s", fetched.FetchTime, stored.FetchTime)
	}
}
