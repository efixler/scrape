package server

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http/httptest"
	nurl "net/url"
	"slices"
	"strconv"
	"strings"
	"testing"

	"github.com/efixler/scrape/fetch"
	"github.com/efixler/scrape/resource"
	"github.com/efixler/scrape/store/sqlite"
)

func TestMutateFeedRequestForBatch(t *testing.T) {
	type data struct {
		url         string
		expectPP    string
		expectOther map[string]string
	}

	tests := []data{
		{"https://foo.com?pp=1&url=http://foo.bar&crunk=X", "1", map[string]string{"crunk": ""}},
	}

	for _, test := range tests {
		var request = httptest.NewRequest("GET", test.url, nil)
		var urls = []string{
			"https://arstechnica.com/?p=1993801",
			"https://arstechnica.com/?p=1993618",
			"https://arstechnica.com/?p=1993507",
			"https://arstechnica.com/?p=1993162",
		}
		mutated := mutateFeedRequestForBatch(request, urls)
		if mutated.Header.Get("Content-Type") != "application/json" {
			t.Errorf("Expected Content-Type 'application/json', got '%s'", mutated.Header.Get("Content-Type"))
		}
		decoder := json.NewDecoder(mutated.Body)
		var batchRequest BatchRequest
		err := decoder.Decode(&batchRequest)
		if err != nil {
			t.Errorf("Error decoding JSON: %s", err)
		}
		fmt.Printf("Form Values %v", mutated.Form)
		if len(batchRequest.Urls) != len(urls) {
			t.Errorf("Expected %d urls, got %d", len(urls), len(batchRequest.Urls))
		}
		if !slices.Equal(batchRequest.Urls, urls) {
			t.Errorf("Expected %v, got %v", urls, batchRequest.Urls)
		}
		if mutated.FormValue("pp") != test.expectPP {
			t.Errorf("Expected PrettyPrint %v, got %v", request.FormValue("pp"), mutated.FormValue("pp"))
		}
		if mutated.FormValue("url") != "" {
			t.Errorf("Expected url to be empty, got %v", mutated.FormValue("url"))
		}
		for k, v := range test.expectOther {
			if mutated.FormValue(k) != v {
				t.Errorf("Expected %s=%s, got %s=%s", k, v, k, mutated.FormValue(k))
			}
		}
	}
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
	scrapeServer := &scrapeServer{feedFetcher: mockFeedFetcher}

	urlBase := "http://foo.bar" // just make the initial URL valid

	for _, test := range tests {
		url := urlBase + test.urlPath
		request := httptest.NewRequest("GET", url, nil)
		w := httptest.NewRecorder()
		scrapeServer.feedHandler(w, request)
		response := w.Result()
		if response.StatusCode != test.expected {
			t.Errorf("Expected status code %d for %s, got %d", test.expected, url, response.StatusCode)
		}
	}
}

func init() {
	// this ensures that any sqlite dbs referenced here are in memory
	sqlite.DefaultDatabase = sqlite.InMemoryDBName
}
