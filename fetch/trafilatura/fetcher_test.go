package trafilatura

import (
	_ "embed"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	nurl "net/url"
	"slices"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/efixler/scrape/fetch"
)

func TestTargetURLErrors(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		errCode, _ := strconv.Atoi(strings.TrimPrefix(r.URL.Path, "/"))
		w.WriteHeader(errCode)
		w.Write([]byte(fmt.Sprintf("Err: %d", errCode)))
	}))
	defer ts.Close()
	client := ts.Client()
	topts := *DefaultOptions
	topts.HttpClient = client
	fetcher := NewTrafilaturaFetcher(topts)
	type data struct {
		url         string
		expectedErr error
	}
	tests := []data{
		{"/400", fetch.ErrHTTPError{StatusCode: http.StatusBadRequest, Message: fmt.Sprintf("Err: %d", http.StatusBadRequest)}},
		{"/401", fetch.ErrHTTPError{StatusCode: http.StatusUnauthorized, Message: fmt.Sprintf("Err: %d", http.StatusUnauthorized)}},
		{"/403", fetch.ErrHTTPError{StatusCode: http.StatusForbidden, Message: fmt.Sprintf("Err: %d", http.StatusForbidden)}},
		{"/404", fetch.ErrHTTPError{StatusCode: http.StatusNotFound, Message: fmt.Sprintf("Err: %d", http.StatusNotFound)}},
		{"/429", fetch.ErrHTTPError{StatusCode: http.StatusTooManyRequests, Message: fmt.Sprintf("Err: %d", http.StatusTooManyRequests)}},
		{"/500", fetch.ErrHTTPError{StatusCode: http.StatusInternalServerError, Message: fmt.Sprintf("Err: %d", http.StatusInternalServerError)}},
		{"/503", fetch.ErrHTTPError{StatusCode: http.StatusServiceUnavailable, Message: fmt.Sprintf("Err: %d", http.StatusServiceUnavailable)}},
	}
	for _, test := range tests {
		url := ts.URL + test.url
		netURL, _ := nurl.Parse(url)
		resource, err := fetcher.Fetch(netURL)
		if err == nil {
			t.Errorf("Expected error for %s", test.url)
		}
		if !errors.Is(err, test.expectedErr) {
			t.Errorf("Expected error %s for %s, got %s", test.expectedErr, test.url, err)
		}
		if resource == nil {
			t.Fatal("Expected resource, got nil")
		}
		if resource.RequestedURL.String() != url {
			t.Errorf("Expected URL %s for resource, got %s", url, resource.RequestedURL.String())
		}
		if resource.FetchTime == nil {
			t.Error("Expected fetch time, got nil")
		}
	}
}

func TestClientFollowsRedirects(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/301":
			http.Redirect(w, r, "/200", http.StatusMovedPermanently)
		case "/302":
			http.Redirect(w, r, "/200", http.StatusFound)
		case "/303":
			http.Redirect(w, r, "/200", http.StatusSeeOther)
		case "/307":
			http.Redirect(w, r, "/200", http.StatusTemporaryRedirect)
		case "/308":
			http.Redirect(w, r, "/200", http.StatusPermanentRedirect)
		case "/200":
			w.Write([]byte("OK"))
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer ts.Close()
	client := ts.Client()
	topts := *DefaultOptions
	topts.HttpClient = client
	fetcher := NewTrafilaturaFetcher(topts)
	type data struct {
		url         string
		expectedErr error
	}
	tests := []data{
		{"/301", nil},
		{"/302", nil},
		{"/303", nil},
		{"/307", nil},
		{"/308", nil},
	}
	for _, test := range tests {
		url := ts.URL + test.url
		netURL, _ := nurl.Parse(url)
		resource, err := fetcher.Fetch(netURL)
		if err != nil {
			t.Errorf("Expected no error for %s, got %s", test.url, err)
		}
		if (resource == nil) || (resource.ContentText != "OK") {
			t.Errorf("Expected 'OK' for %s, got %s", test.url, resource.ContentText)
		}
	}
}

//go:embed smoker.html
var smokeTestPage []byte

func TestMetadataPopulatedSmokeTest(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html")
		w.WriteHeader(http.StatusOK)
		w.Write(smokeTestPage)
	}))
	defer ts.Close()
	client := ts.Client()
	topts := *DefaultOptions
	topts.HttpClient = client
	fetcher := NewTrafilaturaFetcher(topts)
	url := ts.URL + "/0e35649e7413c52ee4502525b548c645.html"
	netURL, _ := nurl.Parse(url)
	resource, err := fetcher.Fetch(netURL)
	if err != nil {
		t.Errorf("Expected no error for %s, got %s", url, err)
	}
	if resource == nil {
		t.Fatal("Expected resource, got nil")
	}
	if resource.RequestedURL.String() != url {
		t.Errorf("Expected URL %s for resource, got %s", url, resource.RequestedURL.String())
	}
	if resource.Title != "Smoke Test" {
		t.Errorf("Expected title 'Smoke Test' for %s, got %s", url, resource.Title)
	}
	if resource.Author != "Joe Blow" {
		t.Errorf("Expected author 'Joe Blow' for %s, got %s", url, resource.Author)
	}
	if resource.Description != "This is a smoke test" {
		t.Errorf("Expected description 'This is a smoke test' for %s, got %s", url, resource.Description)
	}
	if resource.Language != "en" {
		t.Errorf("Expected language 'en' for %s, got %s", url, resource.Language)
	}
	if !slices.Equal(resource.Tags, []string{"test", "smoke"}) {
		t.Errorf("Expected tags 'test, smoke' for %s, got %s", url, resource.Tags)
	}
	if !slices.Equal(resource.Categories, []string{"Cat1", "Cat2"}) {
		t.Errorf("Expected tags 'test, smoke' for %s, got %s", url, resource.Tags)
	}
	if resource.PageType != "article" {
		t.Errorf("Expected page type 'article' for %s, got %s", url, resource.PageType)
	}
	referenceTime := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
	if resource.Date != referenceTime {
		t.Errorf("Expected date %s for %s, got %s", referenceTime, url, resource.Date)
	}
	if resource.Sitename != "smoke.scrape" {
		t.Errorf("Expected site name 'smoke.scrape' for %s, got %s", url, resource.Sitename)
	}
	contentUrl, _ := nurl.Parse("http://smoke.scrape")
	if resource.URL().String() != contentUrl.String() {
		t.Errorf("Expected URL 'http://smoke.scrape' for %s, got %s", url, resource.URL().String())
	}
	if resource.Image != "https://smoke.scrape/image.png" {
		t.Errorf("Expected image 'https://smoke.scrape/image.png' for %s, got %s", url, resource.Image)
	}
	smokerContent := "Smoke Test Smoke test body this is english"
	if resource.ContentText != smokerContent {
		t.Errorf("Expected '%s' for %s, got '%s'", smokerContent, url, resource.ContentText)
	}
}
