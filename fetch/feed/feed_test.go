package feed

import (
	"context"
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
	options := DefaultOptions
	options.Timeout = timeout
	options.Client = client
	fetcher := NewFeedFetcher(options)
	fetcher.Open(context.Background())
	defer fetcher.Close()
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
	options := DefaultOptions
	options.Client = client
	fetcher := NewFeedFetcher(options)
	fetcher.Open(context.Background())
	defer fetcher.Close()
	url, _ := nurl.Parse(ts.URL)
	feed, err := fetcher.Fetch(url)
	if err != nil {
		t.Errorf("Unexpected error for %s: %s", url, err)
	}
	if feed.RequestedURL != url.String() {
		t.Errorf("Expected URL %s, got %s", url, feed.RequestedURL)
	}
}
