package server

import (
	"bytes"
	"context"
	_ "embed"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/pprof"
	nurl "net/url"

	"github.com/efixler/scrape"
	"github.com/efixler/scrape/fetch"
	"github.com/efixler/scrape/fetch/feed"
	"github.com/efixler/scrape/fetch/trafilatura"
	"github.com/efixler/scrape/resource"
	"github.com/efixler/scrape/store/sqlite"
)

func InitMux(ctx context.Context, withProfiling bool) (http.Handler, error) {
	mux := http.NewServeMux()
	mux.HandleFunc("/", handleHome)
	scrapeServer, err := NewScrapeServer(ctx)
	if err != nil {
		return nil, err
	}
	mux.HandleFunc("/extract", scrapeServer.singleHandler)
	mux.HandleFunc("/batch", scrapeServer.batchHandler)
	mux.HandleFunc("/feed", scrapeServer.feedHandler)
	if withProfiling {
		initPProf(mux)
	}
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
	w.WriteHeader(http.StatusOK)
	w.Write(home)
}

func NewScrapeServer(ctx context.Context) (*scrapeServer, error) {
	fetcher, err := scrape.NewStorageBackedFetcher(
		trafilatura.Factory(*trafilatura.DefaultOptions),
		sqlite.Factory(sqlite.DefaultDatabase),
	)
	if err != nil {
		return nil, err
	}
	feedFetcher := feed.NewFeedFetcher(feed.DefaultOptions)
	handler := &scrapeServer{
		fetcher:     fetcher,
		feedFetcher: feedFetcher,
	}
	err = fetcher.Open(ctx)
	if err != nil {
		return nil, err
	}
	err = feedFetcher.Open(ctx)
	if err != nil {
		return nil, err
	}
	return handler, nil
}

// still working out the right way structure this. We probably will want to do
// concurrency by channelizing fetchers. We also don't want to allocate/open
// a new fetcher at every request. For now, we're going to persist one fetcher
// and use the background comtext to ensure it's closed when the server is done
type scrapeServer struct {
	fetcher     *scrape.StorageBackedFetcher
	feedFetcher *feed.FeedFetcher
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
		if errors.Is(err, fetch.ErrUnsupportedContentType) {
			w.WriteHeader(http.StatusUnsupportedMediaType)
		} else {
			w.WriteHeader(http.StatusUnprocessableEntity)
		}
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
	pages := make([]*resource.WebPage, 0, len(req.Urls))
	var page *resource.WebPage
	for _, url := range req.Urls {
		parsedUrl, err := nurl.Parse(url)
		if err != nil {
			page = &resource.WebPage{
				OriginalURL: url,
				Error:       err,
			}
		}
		// In this case we ignore the error, since it'll be included in the page
		page, _ = h.fetcher.Fetch(parsedUrl)
		pages = append(pages, page)
	}
	encoder := json.NewEncoder(w)
	pp := r.FormValue("pp") != ""
	if pp {
		encoder.SetIndent("", "  ")
	}
	err = encoder.Encode(pages)
	if err != nil {
		http.Error(w, fmt.Sprintf("Error encoding response: %s", err), http.StatusInternalServerError)
		return
	}
}

func (h *scrapeServer) feedHandler(w http.ResponseWriter, r *http.Request) {
	url := r.FormValue("url")
	if url == "" {
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte("No URL provided"))
		return
	}
	netUrl, err := nurl.Parse(url)
	//TODO: Use the FetchHTTPError to pass status code and message here
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte(fmt.Sprintf("Invalid URL provided: %q, %s", url, err)))
		return
	}
	resource, err := h.feedFetcher.Fetch(netUrl)
	if err != nil {
		w.WriteHeader(http.StatusUnprocessableEntity)
		w.Write([]byte(fmt.Sprintf("Error fetching %s: %s", url, err)))
		return
	}
	links := resource.ItemLinks()

	batchReq := mutateFeedRequestForBatch(r, links)
	h.batchHandler(w, batchReq)
}

func mutateFeedRequestForBatch(original *http.Request, urls []string) *http.Request {
	r := new(http.Request)
	*r = *original
	//clear the form in the new request
	r.Form = make(nurl.Values)
	//only pass on the pretty print parameter if it's on
	if original.FormValue("pp") == "1" {
		r.Form.Set("pp", "1")
	}
	r.Method = http.MethodPost
	var buffer = &bytes.Buffer{}
	encoder := json.NewEncoder(buffer)
	encoder.Encode(BatchRequest{Urls: urls})
	r.Body = io.NopCloser(buffer)
	r.ContentLength = int64(buffer.Len())
	r.Header.Set("Content-Type", "application/json")
	return r
}

func initPProf(mux *http.ServeMux) {
	// pprof
	mux.HandleFunc("/debug/pprof/", http.HandlerFunc(pprof.Index))
	mux.HandleFunc("/debug/pprof/cmdline", http.HandlerFunc(pprof.Cmdline))
	mux.HandleFunc("/debug/pprof/profile", http.HandlerFunc(pprof.Profile))
	mux.HandleFunc("/debug/pprof/symbol", http.HandlerFunc(pprof.Symbol))
	mux.HandleFunc("/debug/pprof/trace", http.HandlerFunc(pprof.Trace))
}
