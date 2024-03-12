package server

import (
	"bytes"
	"context"
	_ "embed"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/http/pprof"
	nurl "net/url"

	jstream "github.com/efixler/scrape/json"

	"github.com/efixler/scrape"
	"github.com/efixler/scrape/fetch"
	"github.com/efixler/scrape/fetch/feed"
	"github.com/efixler/scrape/fetch/trafilatura"
	"github.com/efixler/scrape/internal/server/healthchecks"
	"github.com/efixler/scrape/resource"
	"github.com/efixler/scrape/store"
)

func InitMux(
	ctx context.Context,
	sf store.Factory,
	headlessRoundTripper http.RoundTripper,
) (*http.ServeMux, error) {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /{$}", handleHome)
	scrapeServer, err := NewScrapeServer(ctx, sf, headlessRoundTripper)
	if err != nil {
		return nil, err
	}
	mux.HandleFunc("GET /extract", scrapeServer.singleHandler())
	mux.HandleFunc("POST /extract", scrapeServer.singleHandler())
	mux.HandleFunc("POST /batch", scrapeServer.batchHandler())
	mux.HandleFunc("/feed", scrapeServer.feedHandler)
	obs, _ := scrapeServer.Storage().(store.Observable)
	mux.Handle("GET /.well-known/", healthchecks.Handler("/.well-known", obs))
	return mux, nil
}

func EnableProfiling(mux *http.ServeMux) {
	initPProf(mux)
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

// When the context passed here is cancelled, the associated fetcher will
// close and release any resources they have open.
func NewScrapeServer(ctx context.Context, sf store.Factory, hrt http.RoundTripper) (*scrapeServer, error) {
	urlFetcher, err := scrape.NewStorageBackedFetcher(
		trafilatura.Factory(),
		sf,
	)
	if err != nil {
		return nil, err
	}
	feedFetcher := feed.NewFeedFetcher(feed.DefaultOptions)
	handler := &scrapeServer{
		urlFetcher:  urlFetcher,
		feedFetcher: feedFetcher,
	}
	err = urlFetcher.Open(ctx)
	if err != nil {
		return nil, err
	}
	if hrt != nil {
		err = handler.makeHeadlessFetcher(ctx, hrt)
		if err != nil {
			slog.Error("Error creating headless fetcher, headless options are disabled", "error", err)
		}
	}
	err = feedFetcher.Open(ctx)
	if err != nil {
		return nil, err
	}
	return handler, nil
}

func (s *scrapeServer) makeHeadlessFetcher(ctx context.Context, ht http.RoundTripper) error {
	hf, err := trafilatura.New(trafilatura.WithTransport(ht))
	if err != nil {
		return err
	}
	if sbf, ok := s.urlFetcher.(*scrape.StorageBackedFetcher); ok {
		s.headlessFetcher, err = sbf.WithAlternateURLFetcher(ctx, hf)
	} else {
		s.headlessFetcher = hf
	}
	return err
}

// The server struct is stateless but uses the same fetcher pair, across all
// requests. These both have some initializaton with and they are also both
// concurrency safe, so if we want to paralellize batch requests we can still
// use this same singelton struct across all requests.
// TODO: The two fetchers should share the same httpClient for their outbound
// requests. Since httpClients pool and resuse connections this would make them
// a little more efficient.
type scrapeServer struct {
	urlFetcher      fetch.URLFetcher
	headlessFetcher fetch.URLFetcher
	feedFetcher     fetch.FeedFetcher
}

// Convenience method to get the underlying storage from the fetcher
// which we use for healthchecks.
// TODO: Re-evaluate. The underlying DB should probably be exposed via an
// interface method, or (less likely) it could be references in the context
// for consumers that we want to keep decoupled.
func (h *scrapeServer) Storage() store.URLDataStore {
	sbf, ok := h.urlFetcher.(*scrape.StorageBackedFetcher)
	if !ok {
		return nil
	}

	return sbf.Storage
}

type singleRequest struct {
	URL         *nurl.URL `json:"url"`
	PrettyPrint bool      `json:"pp,omitempty"`
}

func (h *scrapeServer) singleHandler() http.HandlerFunc {
	return Chain(h.extract, ParseSingle())
}

func (h *scrapeServer) extract(w http.ResponseWriter, r *http.Request) {
	req, _ := r.Context().Value(payloadKey{}).(*singleRequest)
	w.Header().Set("Content-Type", "application/json")
	page, err := h.urlFetcher.Fetch(req.URL)
	if err != nil {
		if errors.Is(err, fetch.HttpError{}) {
			switch err.(fetch.HttpError).StatusCode {
			case http.StatusUnsupportedMediaType:
				fallthrough
			case http.StatusGatewayTimeout:
				w.WriteHeader(err.(fetch.HttpError).StatusCode)
			}
		} else {
			w.WriteHeader(http.StatusUnprocessableEntity)
		}
	}
	encoder := json.NewEncoder(w)
	encoder.SetEscapeHTML(false)
	if req.PrettyPrint {
		encoder.SetIndent("", "  ")
	}
	encoder.Encode(page)
}

type BatchRequest struct {
	Urls []string `json:"urls"`
}

func (h *scrapeServer) batchHandler() http.HandlerFunc {
	return Chain(h.batch, MaxBytes(32768), DecodeJSONBody[BatchRequest]())
}

func (h *scrapeServer) batch(w http.ResponseWriter, r *http.Request) {
	req, ok := r.Context().Value(payloadKey{}).(*BatchRequest)
	if !ok {
		http.Error(w, "No batch request found", http.StatusInternalServerError)
		return
	}
	// maybe should not be an error?
	if len(req.Urls) == 0 {
		http.Error(w, "No URLs provided", http.StatusUnprocessableEntity)
		return
	}
	// if we made it here we are going to return JSON
	w.Header().Set("Content-Type", "application/json")

	encoder := jstream.NewArrayEncoder[*resource.WebPage](w, false)
	pp := r.FormValue("pp") == "1"
	if pp {
		encoder.SetIndent("", "  ")
	}
	var err error
	if batchFetcher, ok := h.urlFetcher.(fetch.BatchURLFetcher); ok {
		rchan := batchFetcher.Batch(req.Urls, fetch.BatchOptions{})
		for page := range rchan {
			err = encoder.Encode(page)
			if err != nil {
				break
			}
		}
	} else { // transitionally while we iron out the throttle-able batch
		h.synchronousBatch(req.Urls, encoder)
	}
	encoder.Finish()
	if err != nil {
		// this error is probably too late to matter, so let's log here:
		slog.Error("Error encoding batch response", "error", err)
	}
}

func (h *scrapeServer) synchronousBatch(urls []string, encoder *jstream.ArrayEncoder[*resource.WebPage]) {
	var page *resource.WebPage
	for _, url := range urls {
		if parsedUrl, err := nurl.Parse(url); err != nil {
			page = &resource.WebPage{
				OriginalURL: url,
				Error:       err,
			}
		} else {
			// In this case we ignore the error, since it'll be included in the page
			page, _ = h.urlFetcher.Fetch(parsedUrl)
		}
		err := encoder.Encode(page)
		if err != nil {
			break
		}
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
		var httpErr fetch.HttpError
		if errors.As(err, &httpErr) {
			w.WriteHeader(httpErr.StatusCode)
			w.Write([]byte(httpErr.Message))
		} else {
			w.WriteHeader(http.StatusUnprocessableEntity)
			w.Write([]byte(fmt.Sprintf("Error fetching %s: %s", url, err)))
		}
		return
	}
	links := resource.ItemLinks()

	batchReq := mutateFeedRequestForBatch(r, links)
	h.batch(w, batchReq)
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
	mux.HandleFunc("GET /debug/pprof/", http.HandlerFunc(pprof.Index))
	mux.HandleFunc("GET /debug/pprof/cmdline", http.HandlerFunc(pprof.Cmdline))
	mux.HandleFunc("GET /debug/pprof/profile", http.HandlerFunc(pprof.Profile))
	mux.HandleFunc("GET /debug/pprof/symbol", http.HandlerFunc(pprof.Symbol))
	mux.HandleFunc("GET /debug/pprof/trace", http.HandlerFunc(pprof.Trace))
}
