package trafilatura

import (
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	nurl "net/url"
	"strconv"
	"strings"
	"testing"

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
		{"/500", fetch.ErrHTTPError{StatusCode: http.StatusInternalServerError, Message: fmt.Sprintf("Err: %d", http.StatusInternalServerError)}},
		{"/503", fetch.ErrHTTPError{StatusCode: http.StatusServiceUnavailable, Message: fmt.Sprintf("Err: %d", http.StatusServiceUnavailable)}},
	}
	for _, test := range tests {
		url := ts.URL + test.url
		netURL, _ := nurl.Parse(url)
		_, err := fetcher.Fetch(netURL)
		if err == nil {
			t.Errorf("Expected error for %s", test.url)
		}
		if !errors.Is(err, test.expectedErr) {
			t.Errorf("Expected error %s for %s, got %s", test.expectedErr, test.url, err)
		}
	}
}
