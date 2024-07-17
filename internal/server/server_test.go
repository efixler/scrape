package server

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	nurl "net/url"
	"strconv"
	"strings"
	"testing"

	"github.com/efixler/scrape/database"
	"github.com/efixler/scrape/database/sqlite"
	"github.com/efixler/scrape/fetch"
	"github.com/efixler/scrape/fetch/trafilatura"
	"github.com/efixler/scrape/internal"
	"github.com/efixler/scrape/internal/storage"
	"github.com/efixler/scrape/resource"
)

type mockUrlFetcher struct {
	fetchMethod resource.FetchClient
}

func (m *mockUrlFetcher) Open(ctx context.Context) error { return nil }
func (m *mockUrlFetcher) Close() error                   { return nil }
func (m *mockUrlFetcher) Fetch(url *nurl.URL) (*resource.WebPage, error) {
	r := &resource.WebPage{
		OriginalURL:  url.String(),
		RequestedURL: url,
		StatusCode:   200,
		ContentText:  "Hello, world!",
		FetchMethod:  m.fetchMethod,
	}

	return r, nil
}

type mockFeedFetcher struct{}

func (m *mockFeedFetcher) Open(ctx context.Context) error { return nil }
func (m *mockFeedFetcher) Close() error                   { return nil }
func (m *mockFeedFetcher) Fetch(url *nurl.URL) (*resource.Feed, error) {
	errCode, atoiErr := strconv.Atoi(strings.TrimPrefix(url.Path, "/"))
	if errCode != 0 {
		return nil, fetch.HttpError{StatusCode: errCode}
	}
	return nil, fmt.Errorf("Error converting %s to int: %s", url.Path, atoiErr)
}

func TestFeedSourceErrors(t *testing.T) {
	type data struct {
		urlPath  string
		expected int
	}
	tests := []data{
		{urlPath: "/", expected: 400},
		{urlPath: "?url=", expected: 400},
		{urlPath: "?url=foo_scheme:invalidurl", expected: 400},
		{urlPath: "?url=http://[::1", expected: 400},
		{urlPath: "/?url=http://passthru.com/400", expected: 400},
		{urlPath: "/?url=http://passthru.com/415", expected: 415},
		{urlPath: "/?url=http://passthru.com/422", expected: 422},
		{urlPath: "/?url=http://passthru.com/508", expected: 508},
	}
	mockFeedFetcher := &mockFeedFetcher{}
	scrapeServer := MustScrapeServer(
		context.Background(),
		WithFeedFetcher(mockFeedFetcher),
		WithURLFetcher(&mockUrlFetcher{}),
	)

	urlBase := "http://foo.bar" // just make the initial URL valid
	handler := scrapeServer.feedHandler()
	for _, test := range tests {
		url := urlBase + test.urlPath
		request := httptest.NewRequest("GET", url, nil)
		w := httptest.NewRecorder()
		handler(w, request)
		response := w.Result()
		if response.StatusCode != test.expected {
			t.Errorf("Expected status code %d for %s, got %d", test.expected, url, response.StatusCode)
		}
	}
}

func TestBatchReponseIsValid(t *testing.T) {
	t.Parallel()
	var dbh = database.New(sqlite.MustNew(sqlite.InMemoryDB()))
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	if err := dbh.Open(ctx); err != nil {
		t.Fatalf("Could not open database: %v", err)
	}

	fetcher := internal.NewStorageBackedFetcher(
		trafilatura.MustNew(nil),
		storage.NewURLDataStore(dbh),
	)

	ss := MustScrapeServer(
		ctx,
		WithURLFetcher(fetcher),
	)

	mux, err := InitMux(ss, nil, false)
	if err != nil {
		t.Fatal(err)
	}
	ts := httptest.NewServer(mux)
	defer ts.Close()
	client := ts.Client()
	urlPath := "/batch"
	targetUrl := ts.URL + urlPath
	var batchPayload BatchRequest
	batchPayload.Urls = []string{
		ts.URL,
		ts.URL + "/1",
		ts.URL + "/2",
	}
	var buf = new(bytes.Buffer)
	payloadEncoder := json.NewEncoder(buf)
	payloadEncoder.Encode(batchPayload)
	resp, err := client.Post(targetUrl, "application/json", buf)
	if err != nil {
		t.Fatal(err)
	}
	if resp.StatusCode != 200 {
		t.Errorf("Expected 200 OK status, got %d (url: %s)", resp.StatusCode, targetUrl)
	}
	if resp.Header.Get("Content-Type") != "application/json" {
		t.Errorf("Expected Content-Type 'application/json', got '%s'", resp.Header.Get("Content-Type"))
	}
	var batchResponse []*resource.WebPage
	decoder := json.NewDecoder(resp.Body)
	err = decoder.Decode(&batchResponse)
	if err != nil {
		t.Errorf("Error decoding JSON: %s", err)
	}
	if len(batchResponse) != len(batchPayload.Urls) {
		t.Fatalf("Expected %d URLs, got %d", len(batchPayload.Urls), len(batchResponse))
	}
	// NB: At this batch we expect the results order to be the same as the
	// input order.
	for i, url := range batchPayload.Urls {
		if batchResponse[i].OriginalURL != url {
			t.Errorf("Expected URL %s, got %s", url, batchResponse[i].OriginalURL)
		}
	}
}

func TestNewFailsWithNilFetcher(t *testing.T) {
	if _, err := NewScrapeServer(context.Background(), WithURLFetcher(nil)); err == nil {
		t.Error("Expected error on nil URLFetcher, got nil")
	}
	if _, err := NewScrapeServer(context.Background()); err == nil {
		t.Error("Expected error on no URLFetcher, got nil")
	}
}

func TestHeadless503WhenUnavailable(t *testing.T) {
	ss := MustScrapeServer(context.Background(), WithURLFetcher(&mockUrlFetcher{}))

	// ss := &scrapeServer{headlessFetcher: nil}
	handler := ss.singleHeadlessHandler()
	req := httptest.NewRequest("GET", "http://foo.bar?url=http://example.com", nil)
	w := httptest.NewRecorder()
	handler(w, req)
	resp := w.Result()
	if resp.StatusCode != 503 {
		t.Errorf("Expected 503, got %d", resp.StatusCode)
	}
}

func TestSingleHandler(t *testing.T) {
	ss := MustScrapeServer(
		context.Background(),
		WithURLFetcher(&mockUrlFetcher{fetchMethod: resource.DefaultClient}),
		WithHeadlessIf(&mockUrlFetcher{fetchMethod: resource.HeadlessChromium}),
	)
	tests := []struct {
		name         string
		url          string
		handler      http.HandlerFunc
		expectMethod resource.FetchClient
	}{
		{
			name:         "client",
			url:          "http://foo.bar",
			handler:      ss.singleHandler(),
			expectMethod: resource.DefaultClient,
		},
		{
			name:         "headless",
			url:          "http://example.com",
			handler:      ss.singleHeadlessHandler(),
			expectMethod: resource.HeadlessChromium,
		},
	}

	for _, test := range tests {
		req := httptest.NewRequest("GET", "http://foo.bar?url="+test.url, nil)
		w := httptest.NewRecorder()
		test.handler(w, req)
		resp := w.Result()
		if resp.StatusCode != 200 {
			t.Errorf("[%s] Expected 200, got %d", test.name, resp.StatusCode)
		}
		decoder := json.NewDecoder(resp.Body)
		decoder.DisallowUnknownFields()
		var r resource.WebPage
		err := decoder.Decode(&r)
		if err != nil {
			t.Fatalf("[%s] Error decoding JSON: %s", test.name, err)
		}
		if r.OriginalURL != test.url {
			t.Errorf("[%s] Expected URL %s, got %s", test.name, test.url, r.OriginalURL)
		}
		if r.StatusCode != 200 {
			t.Errorf("[%s] Expected status code 200, got %d", test.name, r.StatusCode)
		}
		if r.ContentText != "Hello, world!" {
			t.Errorf("[%s] Expected 'Hello, world!', got '%s'", test.name, r.ContentText)
		}
		if r.FetchMethod != test.expectMethod {
			t.Errorf("[%s] Expected fetch method %s, got %s", test.name, test.expectMethod, r.FetchMethod)
		}
	}
}

func TestDeleteHandler(t *testing.T) {
	ss := MustScrapeServer(
		context.Background(),
		WithURLFetcher(&mockUrlFetcher{}),
	)
	tests := []struct {
		name           string
		body           string
		expectedResult int
	}{
		{
			name:           "no body",
			body:           "",
			expectedResult: 400,
		},
		{
			name:           "good body, bad handler",
			body:           "{\"url\":\"http://foo.bar\"}",
			expectedResult: 501,
		},
		{
			name:           "bad body params",
			body:           "{\"foobar\":\"bar\"}",
			expectedResult: 400,
		},
		// The handler is current bound to the concrete StorageBackedFetcher
		// need to fix this so we can mock a handler that will actually do a delete
	}

	for _, test := range tests {
		req := httptest.NewRequest("DELETE", "http://foo.bar", strings.NewReader(test.body))
		w := httptest.NewRecorder()
		ss.deleteHandler()(w, req)
		resp := w.Result()
		if resp.StatusCode != test.expectedResult {
			t.Errorf("[%s] Expected %d, got %d", test.name, test.expectedResult, resp.StatusCode)
		}
	}
}
