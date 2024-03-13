package server

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http/httptest"
	nurl "net/url"
	"strconv"
	"strings"
	"testing"

	"github.com/efixler/scrape/fetch"
	"github.com/efixler/scrape/internal/storage/sqlite"
	"github.com/efixler/scrape/resource"
)

var storeFactory = sqlite.Factory(sqlite.InMemoryDB())

// func TestMutateFeedRequestForBatch(t *testing.T) {
// 	type data struct {
// 		url         string
// 		expectPP    string
// 		expectOther map[string]string
// 	}

// 	tests := []data{
// 		{"https://foo.com?pp=1&url=http://foo.bar&crunk=X", "1", map[string]string{"crunk": ""}},
// 	}

// 	for _, test := range tests {
// 		var request = httptest.NewRequest("GET", test.url, nil)
// 		var urls = []string{
// 			"https://arstechnica.com/?p=1993801",
// 			"https://arstechnica.com/?p=1993618",
// 			"https://arstechnica.com/?p=1993507",
// 			"https://arstechnica.com/?p=1993162",
// 		}
// 		mutated := mutateFeedRequestForBatch(request, urls)
// 		if mutated.Header.Get("Content-Type") != "application/json" {
// 			t.Errorf("Expected Content-Type 'application/json', got '%s'", mutated.Header.Get("Content-Type"))
// 		}
// 		decoder := json.NewDecoder(mutated.Body)
// 		var batchRequest BatchRequest
// 		err := decoder.Decode(&batchRequest)
// 		if err != nil {
// 			t.Errorf("Error decoding JSON: %s", err)
// 		}
// 		if len(batchRequest.Urls) != len(urls) {
// 			t.Errorf("Expected %d urls, got %d", len(urls), len(batchRequest.Urls))
// 		}
// 		if !slices.Equal(batchRequest.Urls, urls) {
// 			t.Errorf("Expected %v, got %v", urls, batchRequest.Urls)
// 		}
// 		if mutated.FormValue("pp") != test.expectPP {
// 			t.Errorf("Expected PrettyPrint %v, got %v", request.FormValue("pp"), mutated.FormValue("pp"))
// 		}
// 		if mutated.FormValue("url") != "" {
// 			t.Errorf("Expected url to be empty, got %v", mutated.FormValue("url"))
// 		}
// 		for k, v := range test.expectOther {
// 			if mutated.FormValue(k) != v {
// 				t.Errorf("Expected %s=%s, got %s=%s", k, v, k, mutated.FormValue(k))
// 			}
// 		}
// 	}
// }

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

func TestWellknown(t *testing.T) {
	t.Parallel()
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	mux, err := InitMux(ctx, storeFactory, nil)
	if err != nil {
		t.Fatal(err)
	}
	ts := httptest.NewServer(mux)
	defer ts.Close()
	client := ts.Client()
	urlPath := "/.well-known/heartbeat"
	targetUrl := ts.URL + urlPath
	resp, err := client.Get(targetUrl)
	if err != nil {
		t.Fatal(err)
	}
	if resp.StatusCode != 200 {
		t.Errorf("Expected 200 OK, got %d (url: %s)", resp.StatusCode, targetUrl)
	}
	urlPath = "/.well-known/health"
	targetUrl = ts.URL + urlPath
	resp, err = client.Get(targetUrl)
	if err != nil {
		t.Fatal(err)
	}
	if resp.StatusCode != 200 {
		t.Errorf("Expected 200 OK, got %d (url: %s)", resp.StatusCode, targetUrl)
	}
}

func TestBatchReponseIsValid(t *testing.T) {
	t.Parallel()
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	mux, err := InitMux(ctx, storeFactory, nil)
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

func TestExtractErrors(t *testing.T) {
	t.Parallel()
	type data struct {
		url            string
		expectedStatus int
	}
	tests := []data{
		{url: "/", expectedStatus: 404},
		{url: "", expectedStatus: 400},
		{url: "?url=", expectedStatus: 400},
		{url: "?url=foo_scheme:invalidurl", expectedStatus: 400},
		{url: "?url=http://[::1", expectedStatus: 400},
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	mux, err := InitMux(ctx, storeFactory, nil)
	if err != nil {
		t.Fatal(err)
	}
	ts := httptest.NewServer(mux)
	defer ts.Close()
	client := ts.Client()
	urlPath := "/extract"
	targetUrl := ts.URL + urlPath
	for i, test := range tests {
		resp, err := client.Get(targetUrl + test.url)
		if err != nil {
			t.Fatal(err)
		}
		if resp.StatusCode != test.expectedStatus {
			t.Errorf("Expected %d status code for test %d, got %d", test.expectedStatus, i, resp.StatusCode)
		}
	}
}
