package server

import (
	"context"
	_ "embed"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"net/http/pprof"
	nurl "net/url"

	"github.com/efixler/webutil/jsonarray"

	"github.com/efixler/scrape"
	"github.com/efixler/scrape/fetch"
	"github.com/efixler/scrape/fetch/feed"
	"github.com/efixler/scrape/fetch/trafilatura"
	"github.com/efixler/scrape/internal/server/healthchecks"
	"github.com/efixler/scrape/resource"
	"github.com/efixler/scrape/store"
)

func InitMux(
	scrapeServer *scrapeServer,
) (*http.ServeMux, error) {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /{$}", handleHome)
	// scrapeServer, err := NewScrapeServer(ctx, sf, headlessClient)
	// if err != nil {
	// 	return nil, err
	// }
	h := scrapeServer.singleHandler()
	mux.HandleFunc("GET /extract", h)
	mux.HandleFunc("POST /extract", h)
	h = scrapeServer.singleHeadlessHandler()
	mux.HandleFunc("GET /extract/headless", h)
	mux.HandleFunc("POST /extract/headless", h)
	mux.HandleFunc("POST /batch", scrapeServer.batchHandler())
	mux.HandleFunc("DELETE /{$}", scrapeServer.deleteHandler())
	h = scrapeServer.feedHandler()
	mux.HandleFunc("GET /feed", h)
	mux.HandleFunc("POST /feed", h)
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
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	w.Write(home)
}

// The server struct is stateless but uses the same fetchers across all requests,
// to optimize client and database connections. There's a general fetcher, and
// special ones for headless scrapes and RSS/Atom feeds.
type scrapeServer struct {
	urlFetcher      fetch.URLFetcher
	headlessFetcher fetch.URLFetcher
	feedFetcher     fetch.FeedFetcher
}

// When the context passed here is cancelled, the associated fetcher will
// close and release any resources they have open.
func NewScrapeServer(
	ctx context.Context,
	sf store.Factory,
	clientFactory fetch.Factory,
	headlessFetcher fetch.URLFetcher,
	// headlessClient fetch.Client,
) (*scrapeServer, error) {
	urlFetcher, err := scrape.NewStorageBackedFetcher(
		trafilatura.Factory(nil),
		sf,
	)
	if err != nil {
		return nil, err
	}
	feedFetcher := feed.NewFeedFetcher(feed.DefaultOptions)
	handler := &scrapeServer{
		urlFetcher:      urlFetcher,
		feedFetcher:     feedFetcher,
		headlessFetcher: headlessFetcher,
	}
	err = handler.urlFetcher.Open(ctx)
	if err != nil {
		return nil, err
	}

	err = feedFetcher.Open(ctx)
	if err != nil {
		return nil, err
	}
	return handler, nil
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

func (h *scrapeServer) singleHandler() http.HandlerFunc {
	return Chain(h.extract, MaxBytes(4096), parseSinglePayload())
}

func (h *scrapeServer) singleHeadlessHandler() http.HandlerFunc {
	return Chain(
		extractWithFetcher(h.headlessFetcher),
		MaxBytes(4096),
		parseSinglePayload(),
	)
}

// The nested handler here is the same as the one below, just enclosed around a fetcher.
// This is here temporarily while experimenting with how to handle headless-variant requests.
// The enclosed approach is tighter than using the context to carry the fetcher, but this won't
// work for feed requests right now because of how they call h.batch at the end (which is malleable).
// Right now headless-variant requests have their own endpoint; if we we to using a payload param
// to choose headless, that'll drive moving a context-based solution for fetcher stashing.
func extractWithFetcher(fetcher fetch.URLFetcher) http.HandlerFunc {
	if fetcher == nil {
		return func(w http.ResponseWriter, r *http.Request) {
			http.Error(w, http.StatusText(http.StatusServiceUnavailable), http.StatusServiceUnavailable)
		}
	}
	return func(w http.ResponseWriter, r *http.Request) {
		req, ok := r.Context().Value(payloadKey{}).(*singleURLRequest)
		if !ok {
			http.Error(w, "Can't process extract request, no input data", http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		page, err := fetcher.Fetch(req.URL)
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
		page.FetchMethod = resource.Headless
		encoder := json.NewEncoder(w)
		encoder.SetEscapeHTML(false)
		if req.PrettyPrint {
			encoder.SetIndent("", "  ")
		}
		encoder.Encode(page)
	}
}

func (h *scrapeServer) extract(w http.ResponseWriter, r *http.Request) {
	req, ok := r.Context().Value(payloadKey{}).(*singleURLRequest)
	if !ok {
		http.Error(w, "Can't process extract request, no input data", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	// fetcher, _ := r.Context().Value(fetcherKey{}).(fetch.URLFetcher)

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

	encoder := jsonarray.NewEncoder[*resource.WebPage](w, false)
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

func (h scrapeServer) deleteHandler() http.HandlerFunc {
	return Chain(h.delete, MaxBytes(4096), parseSinglePayload())
}

func (h scrapeServer) delete(w http.ResponseWriter, r *http.Request) {
	req, ok := r.Context().Value(payloadKey{}).(*singleURLRequest)
	if !ok {
		http.Error(w, "Can't process delete request, no input data", http.StatusInternalServerError)
		return
	}
	deleter, ok := h.urlFetcher.(*scrape.StorageBackedFetcher)
	if !ok {
		http.Error(w, "Can't delete in the current configuration", http.StatusNotImplemented)
		return
	}
	deleted, err := deleter.Delete(req.URL)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	if !deleted {
		http.Error(w, "Not found", http.StatusGone)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *scrapeServer) synchronousBatch(urls []string, encoder *jsonarray.Encoder[*resource.WebPage]) {
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

func (h *scrapeServer) feedHandler() http.HandlerFunc {
	return Chain(h.feed, MaxBytes(4096), parseSinglePayload())
}

func (h *scrapeServer) feed(w http.ResponseWriter, r *http.Request) {
	req, ok := r.Context().Value(payloadKey{}).(*singleURLRequest)
	if !ok {
		http.Error(w, "Can't process extract request, no input data", http.StatusInternalServerError)
		return
	}
	resource, err := h.feedFetcher.Fetch(req.URL)
	if err != nil {
		var httpErr fetch.HttpError
		if errors.As(err, &httpErr) {
			w.WriteHeader(httpErr.StatusCode)
			w.Write([]byte(httpErr.Message))
		} else {
			w.WriteHeader(http.StatusUnprocessableEntity)
			w.Write([]byte(fmt.Sprintf("Error fetching %s: %s", req.URL, err)))
		}
		return
	}
	links := resource.ItemLinks()
	v := BatchRequest{Urls: links}
	r = r.WithContext(context.WithValue(r.Context(), payloadKey{}, &v))
	h.batch(w, r)
}

func initPProf(mux *http.ServeMux) {
	// pprof
	mux.HandleFunc("GET /debug/pprof/", http.HandlerFunc(pprof.Index))
	mux.HandleFunc("GET /debug/pprof/cmdline", http.HandlerFunc(pprof.Cmdline))
	mux.HandleFunc("GET /debug/pprof/profile", http.HandlerFunc(pprof.Profile))
	mux.HandleFunc("GET /debug/pprof/symbol", http.HandlerFunc(pprof.Symbol))
	mux.HandleFunc("GET /debug/pprof/trace", http.HandlerFunc(pprof.Trace))
}
