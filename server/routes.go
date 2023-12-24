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
	"github.com/efixler/scrape/resource"
	"github.com/efixler/scrape/store/sqlite"
)

func InitMux(ctx context.Context) (http.Handler, error) {
	mux := http.NewServeMux()
	mux.HandleFunc("/", handleHome)
	scrapeServer, err := NewScrapeServer(ctx)
	if err != nil {
		return nil, err
	}
	mux.HandleFunc("/extract", scrapeServer.singleHandler)
	mux.HandleFunc("/batch", scrapeServer.batchHandler)
	return mux, nil
}

//go:embed pages/index.html
var home []byte

func handleHome(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/" {
		http.NotFound(w, r)
		return
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Write(home)
}

func NewScrapeServer(ctx context.Context) (*scrapeServer, error) {
	fetcher, err := scrape.NewStorageBackedFetcher(
		trafilatura.Factory(),
		sqlite.Factory(sqlite.DEFAULT_DB_FILENAME),
	)
	if err != nil {
		return nil, err
	}
	handler := &scrapeServer{
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
type scrapeServer struct {
	fetcher *scrape.StorageBackedFetcher
}

func (h *scrapeServer) singleHandler(w http.ResponseWriter, r *http.Request) {
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

type BatchRequest struct {
	Urls []string `json:"urls"`
}

type urlError struct {
	URL   string `json:"URL"`
	Error string `json:"Error"`
}

type batchResponse struct {
	Pages  []*resource.WebPage `json:"Pages"`
	Errors []urlError          `json:"Errors"`
}

func (h *scrapeServer) batchHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Only POST is supported", http.StatusMethodNotAllowed)
		return
	}
	r.Body = http.MaxBytesReader(w, r.Body, 1048576)
	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()

	var req BatchRequest
	err := decoder.Decode(&req)
	if !assertDecode(err, w) {
		return
	}
	// maybe should not be an error?
	if len(req.Urls) == 0 {
		http.Error(w, "No URLs provided", http.StatusUnprocessableEntity)
		return
	}

	//var errors []error = make([]error, 0, len(req.Urls)/20)
	batchResponse := &batchResponse{
		Pages:  make([]*resource.WebPage, 0, len(req.Urls)),
		Errors: make([]urlError, 0, len(req.Urls)/20),
	}
	for _, url := range req.Urls {
		parsedUrl, err := nurl.Parse(url)
		if err != nil {
			batchResponse.Errors = append(batchResponse.Errors, urlError{URL: url, Error: err.Error()})
			continue
		}
		page, err := h.fetcher.Fetch(parsedUrl)
		if err != nil {
			batchResponse.Errors = append(batchResponse.Errors, urlError{URL: url, Error: err.Error()})
			continue
		}
		batchResponse.Pages = append(batchResponse.Pages, page)
	}
	encoder := json.NewEncoder(w)
	pp := r.FormValue("pp") != ""
	if pp {
		encoder.SetIndent("", "  ")
	}
	err = encoder.Encode(batchResponse)
	if err != nil {
		http.Error(w, fmt.Sprintf("Error encoding response: %s", err), http.StatusInternalServerError)
		return
	}
}
