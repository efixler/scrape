package server

import (
	"context"
	_ "embed"
	"encoding/json"
	"fmt"
	"net/http"
	nurl "net/url"

	"github.com/efixler/scrape"
	"github.com/efixler/scrape/fetch/trafilatura"
	"github.com/efixler/scrape/store/sqlite"
)

func InitMux(ctx context.Context) (*http.ServeMux, error) {
	mux := http.NewServeMux()
	mux.HandleFunc("/", handleHome)
	extractHandler, err := NewScrapeHandler(ctx)
	if err != nil {
		return nil, err
	}
	mux.Handle("/extract", extractHandler)
	return mux, nil
}

//go:embed pages/index.html
var home []byte

func handleHome(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Write(home)
}

func NewScrapeHandler(ctx context.Context) (*scrapeHandler, error) {
	fetcher, err := scrape.NewStorageBackedFetcher(
		trafilatura.Factory(),
		sqlite.Factory(sqlite.DEFAULT_DB_FILENAME),
	)
	if err != nil {
		return nil, err
	}
	handler := &scrapeHandler{
		fetcher: fetcher,
	}
	err = fetcher.Open(ctx)
	if err != nil {
		return nil, err
	}
	// this is context we'll use for closing the db resource
	return handler, nil
}

// still working out the right way structure this. We probably will want to do
// concurrency by channelizing fetchers. We also don't want to allocate/open
// a new fetcher at every request. For now, we're going to persist one fetcher
// and use the background comtext to ensure it's closed when the server is done
type scrapeHandler struct {
	fetcher *scrape.StorageBackedFetcher
}

func (h *scrapeHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	url := r.FormValue("url")
	if url == "" {
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte("No URL provided"))
		return
	}
	netUrl, err := nurl.Parse(url)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte(fmt.Sprintf("Invalid URL provided: %q, %s", url, err)))
		return
	}
	page, err := h.fetcher.Fetch(netUrl)
	if err != nil {
		w.WriteHeader(http.StatusUnprocessableEntity)
		w.Write([]byte(fmt.Sprintf("Error fetching %s: %s", url, err)))
		return
	}
	encoder := json.NewEncoder(w)
	pp := r.FormValue("pp") != ""
	if pp {
		encoder.SetIndent("", "  ")
	}
	encoder.Encode(page)
}
