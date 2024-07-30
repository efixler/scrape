package server

import (
	"context"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/efixler/scrape/fetch/trafilatura"
	"github.com/efixler/scrape/internal/auth"
)

func TestWellknown(t *testing.T) {
	t.Parallel()
	//ctx, cancel := context.WithCancel(context.Background())
	//defer cancel()

	mux, err := InitMux(&scrapeServer{}, nil, false, false)
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

func TestExtractErrors(t *testing.T) {
	t.Parallel()
	type data struct {
		url            string
		expectedStatus int
	}
	tests := []data{
		// {url: "/", expectedStatus: 404}, TODO: this is a bug, figure out why it's not 404
		{url: "", expectedStatus: 400},
		{url: "?url=", expectedStatus: 400},
		{url: "?url=foo_scheme:invalidurl", expectedStatus: 400},
		{url: "?url=http://[::1", expectedStatus: 400},
	}

	ss := MustScrapeServer(
		context.Background(),
		WithURLFetcher(trafilatura.MustNew(nil)),
	)

	mux, err := InitMux(ss, nil, false, false)
	if err != nil {
		t.Fatal(err)
	}
	ts := httptest.NewServer(mux)
	defer ts.Close()
	client := ts.Client()
	urlPath := "/extract"
	baseUrl := ts.URL + urlPath
	for i, test := range tests {
		targetUrl := baseUrl + test.url
		resp, err := client.Get(targetUrl)
		if err != nil {
			t.Fatal(err)
		}
		if resp.StatusCode != test.expectedStatus {
			t.Errorf("Expected %d status code for test %d (%s), got %d", test.expectedStatus, i, targetUrl, resp.StatusCode)
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
	ss := MustScrapeServer(
		context.Background(),
		WithURLFetcher(&mockUrlFetcher{}),
		WithAuthorizationIf(auth.MustNewHS256SigningKey()),
	)
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

func init() {
	slog.SetLogLoggerLevel(slog.LevelError)
}
