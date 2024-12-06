package feed

import (
	"errors"
	"net/http"
	"net/http/httptest"
	nurl "net/url"
	"testing"
	"time"

	"github.com/efixler/scrape/fetch"
)

const dummyRSS = `<?xml version="1.0" encoding="UTF-8"?>
<rss version="2.0">
		<channel>
		<title>Example Feed</title><link>http://example.org/</link><description>description</description>
		<lastBuildDate>Mon, 06 Sep 2021 16:45:00 GMT</lastBuildDate>
		<docs>http://www.rssboard.org/rss-specification</docs>
		<item>
		</channel></rss>
`

func TestFetchCancelsOnTimeout(t *testing.T) {
	timeout := 50 * time.Millisecond
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(2 * timeout)
		w.WriteHeader(http.StatusOK)
		w.Header().Set("Content-Type", "application/rss+xml")
		w.Write([]byte(dummyRSS))
	}))
	defer ts.Close()
	client := ts.Client()
	fetcher := MustFeedFetcher(
		WithTimeout(timeout),
		WithClient(client),
	)
	url, _ := nurl.Parse(ts.URL)
	_, err := fetcher.Fetch(url)
	if err == nil {
		t.Errorf("Expected error for %s, got nil", url)
	} else if !errors.Is(err, fetch.HttpError{}) {
		t.Errorf("Expected fetch.HttpError for %s, got %s", url, err)
	} else {
		var httpErr fetch.HttpError
		errors.As(err, &httpErr)
		if httpErr.StatusCode != http.StatusGatewayTimeout {
			t.Errorf("Expected http.StatusGatewayTimeout for %s, got %d", url, httpErr.StatusCode)
		}
	}

}

func TestFetchReturnsRequestedURL(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Header().Set("Content-Type", "application/rss+xml")
		w.Write([]byte(dummyRSS))
	}))
	defer ts.Close()
	client := ts.Client()
	fetcher := MustFeedFetcher(
		WithClient(client),
	)
	url, _ := nurl.Parse(ts.URL)
	feed, err := fetcher.Fetch(url)
	if err != nil {
		t.Errorf("Unexpected error for %s: %s", url, err)
	}
	if feed.RequestedURL != url.String() {
		t.Errorf("Expected URL %s, got %s", url, feed.RequestedURL)
	}
}

func TestWithTimeout(t *testing.T) {
	tests := []struct {
		name      string
		timeout   time.Duration
		expectErr bool
	}{
		{
			name:      "valid",
			timeout:   50 * time.Millisecond,
			expectErr: false,
		},
		{
			name:      "negative",
			timeout:   -1 * time.Millisecond,
			expectErr: true,
		},
		{
			name:      "zero",
			timeout:   0,
			expectErr: true,
		},
	}

	for _, tt := range tests {
		err := WithTimeout(tt.timeout)(&config{})
		if tt.expectErr && err == nil {
			t.Errorf("Expected error for %s, got nil", tt.timeout)
		} else if !tt.expectErr && err != nil {
			t.Errorf("Unexpected error for %s: %s", tt.timeout, err)
		}
	}
}

func TestWithUserAgentOption(t *testing.T) {
	tests := []struct {
		name      string
		ua        string
		expectErr bool
	}{
		{
			name:      "valid",
			ua:        "test",
			expectErr: false,
		},
		{
			name:      "empty",
			ua:        "",
			expectErr: true,
		},
	}

	for _, tt := range tests {
		err := WithUserAgent(tt.ua)(&config{})
		if tt.expectErr && err == nil {
			t.Errorf("[%s] Expected error for %s, got nil", tt.name, tt.ua)
		} else if !tt.expectErr && err != nil {
			t.Errorf("[%s] Unexpected error for %s: %s", tt.name, tt.ua, err)
		}
	}
}

func TestUserAgent(t *testing.T) {
	tests := []struct {
		name     string
		option   option
		expected string
	}{
		{
			name:     "default",
			option:   nil,
			expected: fetch.DefaultUserAgent,
		},
		{
			name:     "custom",
			option:   WithUserAgent("test/1.0"),
			expected: "test/1.0",
		},
	}
	for _, tt := range tests {
		ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.UserAgent() != tt.expected {
				t.Errorf("[%s] Expected %s, got %s", tt.name, tt.expected, r.UserAgent())
			}
			w.WriteHeader(http.StatusOK)
			w.Header().Set("Content-Type", "application/rss+xml")
			w.Write([]byte(dummyRSS))
		}))
		t.Cleanup(ts.Close)
		client := ts.Client()
		options := []option{WithClient(client)}
		if tt.option != nil {
			options = append(options, tt.option)
		}
		fetcher := MustFeedFetcher(options...)
		url, _ := nurl.Parse(ts.URL)
		if _, err := fetcher.Fetch(url); err != nil {
			t.Errorf("[%s] Unexpected error for %s: %s", tt.name, url, err)
		}
	}
}
