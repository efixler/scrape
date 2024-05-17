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

	"github.com/efixler/scrape/fetch"
	"github.com/efixler/scrape/fetch/trafilatura"
	"github.com/efixler/scrape/internal/auth"
	"github.com/efixler/scrape/internal/storage/sqlite"
	"github.com/efixler/scrape/resource"
)

var storeFactory = sqlite.Factory(sqlite.InMemoryDB())

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
	//ctx, cancel := context.WithCancel(context.Background())
	//defer cancel()

	mux, err := InitMux(&scrapeServer{})
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
	ss, _ := NewScrapeServer(ctx, storeFactory, trafilatura.Factory(nil), nil)
	mux, err := InitMux(ss)
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
	ss, _ := NewScrapeServer(ctx, storeFactory, trafilatura.Factory(nil), nil)
	mux, err := InitMux(ss)
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

func TestHeadless503WhenUnavailable(t *testing.T) {
	ss := &scrapeServer{headlessFetcher: nil}
	handler := ss.singleHeadlessHandler()
	req := httptest.NewRequest("GET", "http://foo.bar?url=http://example.com", nil)
	w := httptest.NewRecorder()
	handler(w, req)
	resp := w.Result()
	if resp.StatusCode != 503 {
		t.Errorf("Expected 503, got %d", resp.StatusCode)
	}
}

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

func TestSingleHandler(t *testing.T) {
	ss := &scrapeServer{
		urlFetcher:      &mockUrlFetcher{fetchMethod: resource.DefaultClient},
		headlessFetcher: &mockUrlFetcher{fetchMethod: resource.HeadlessChrome},
	}
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
			expectMethod: resource.HeadlessChrome,
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
	ss := &scrapeServer{urlFetcher: &mockUrlFetcher{}}
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

func TestHomeHandler(t *testing.T) {
	tests := []struct {
		name           string
		key            auth.HMACBase64Key
		expectedResult int
	}{
		{
			name:           "no key",
			key:            nil,
			expectedResult: 200,
		},
		{
			name:           "good key",
			key:            auth.MustNewHS256SigningKey(),
			expectedResult: 200,
		},
	}
	for _, test := range tests {
		ss := &scrapeServer{SigningKey: test.key}
		req := httptest.NewRequest("GET", "http://foo.bar/", nil)
		w := httptest.NewRecorder()
		ss.homeHandler()(w, req)
		resp := w.Result()
		if resp.StatusCode != test.expectedResult {
			t.Errorf("[%s] Expected %d, got %d", test.name, test.expectedResult, resp.StatusCode)
		}
	}
}

func TestMustTemplate(t *testing.T) {
	tests := []struct {
		name        string
		key         auth.HMACBase64Key
		expectToken bool
	}{
		{
			name:        "with key",
			key:         auth.MustNewHS256SigningKey(),
			expectToken: true,
		},
		{
			name:        "no key",
			key:         nil,
			expectToken: false,
		},
		{
			name:        "empty key",
			key:         auth.HMACBase64Key([]byte{}),
			expectToken: false,
		},
	}
	for _, test := range tests {
		ss := &scrapeServer{SigningKey: test.key}
		tmpl := ss.mustHomeTemplate()
		tmpl, err := tmpl.Parse("{{AuthToken}}")
		if err != nil {
			t.Fatalf("[%s] Error parsing template: %s", test.name, err)
		}
		var buf bytes.Buffer
		err = tmpl.Execute(&buf, nil)
		if err != nil {
			t.Fatalf("[%s] Error executing template: %s", test.name, err)
		}
		output := buf.String()
		if !test.expectToken && output != "" {
			t.Fatalf("[%s] Expected empty output, got %s", test.name, output)
		}
		if test.expectToken {
			switch output {
			case "":
				t.Fatalf("[%s] Expected non-empty token, got empty", test.name)
			default:
				_, err := auth.VerifyToken(test.key, output)
				if err != nil {
					t.Fatalf("[%s] Error verifying token: %s", test.name, err)
				}
			}
		}
	}
}

// Test that a request to the relevant API routes without a valid token
// is rejected when running with a signing key.
// Since the auth middleware is (and always should be) placed in the
// middleware chain before the actual handler, we don't need to set up
// a request body - the request should get rejected before that would get
// evaluated.
func TestAPIRoutesAreProtected(t *testing.T) {
	ss := &scrapeServer{SigningKey: auth.MustNewHS256SigningKey()}
	tests := []struct {
		name    string
		method  string
		handler func() http.HandlerFunc
	}{
		{
			name:    "POST /extract",
			method:  http.MethodPost,
			handler: ss.singleHandler,
		},
		{
			name:    "GET /extract",
			method:  http.MethodGet,
			handler: ss.singleHandler,
		},
		{
			name:    "POST /extract/headless",
			method:  http.MethodPost,
			handler: ss.singleHeadlessHandler,
		},
		{
			name:    "POST /extract/batch",
			method:  http.MethodPost,
			handler: ss.batchHandler,
		},
		{
			name:    "DELETE /extract",
			method:  http.MethodDelete,
			handler: ss.deleteHandler,
		},
		{
			name:    "GET /feed",
			method:  http.MethodGet,
			handler: ss.feedHandler,
		},
		{
			name:    "POST /feed",
			method:  http.MethodPost,
			handler: ss.feedHandler,
		},
	}
	for _, test := range tests {
		req := httptest.NewRequest(test.method, "http://foo.bar", nil)
		w := httptest.NewRecorder()
		test.handler()(w, req)
		resp := w.Result()
		if resp.StatusCode != 401 {
			t.Fatalf("[%s] Expected 401, got %d", test.name, resp.StatusCode)
		}
	}
}
