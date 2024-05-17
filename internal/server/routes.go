package server

import (
	"bytes"
	"context"
	"embed"
	"encoding/json"
	"errors"
	"fmt"
	"html/template"
	"log/slog"
	"net/http"
	"net/http/pprof"
	nurl "net/url"
	"time"

	"github.com/efixler/webutil/jsonarray"

	"github.com/efixler/scrape"
	"github.com/efixler/scrape/fetch"
	"github.com/efixler/scrape/fetch/feed"
	"github.com/efixler/scrape/fetch/trafilatura"
	"github.com/efixler/scrape/internal/auth"
	"github.com/efixler/scrape/internal/server/healthchecks"
	"github.com/efixler/scrape/resource"
	"github.com/efixler/scrape/store"
)

func InitMux(scrapeServer *scrapeServer) (*http.ServeMux, error) {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /{$}", scrapeServer.homeHandler())
	h := scrapeServer.singleHandler()
	mux.HandleFunc("GET /extract", h)
	mux.HandleFunc("POST /extract", h)
	h = scrapeServer.singleHeadlessHandler()
	mux.HandleFunc("GET /extract/headless", h)
	mux.HandleFunc("POST /extract/headless", h)
	mux.HandleFunc("POST /batch", scrapeServer.batchHandler())
	mux.HandleFunc("DELETE /extract", scrapeServer.deleteHandler())
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

// The server struct is stateless but uses the same fetchers across all requests,
// to optimize client and database connections. There's a general fetcher, and
// special ones for headless scrapes and RSS/Atom feeds.
type scrapeServer struct {
	urlFetcher      fetch.URLFetcher
	headlessFetcher fetch.URLFetcher
	feedFetcher     fetch.FeedFetcher
	SigningKey      auth.HMACBase64Key
}

// When the context passed here is cancelled, the associated fetcher will
// close and release any resources they have open.
func NewScrapeServer(
	ctx context.Context,
	sf store.Factory,
	clientFactory fetch.Factory,
	headlessFetcher fetch.URLFetcher,
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

type claimsKey struct{}

// Prepend the authorization checker to the list of passed middleware if authorization is enabled.
func (ss scrapeServer) withAuthIfEnabled(ms ...middleware) []middleware {
	if len(ss.SigningKey) > 0 {
		ms = append([]middleware{auth.JWTAuthMiddleware(ss.SigningKey, claimsKey{})}, ms...)
	}
	return ms
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

//go:embed pages/index.html
var home embed.FS

func (h scrapeServer) mustHomeTemplate() *template.Template {
	tmpl := template.New("home")
	var keyF = func() string { return "" }
	if len(h.SigningKey) > 0 {
		keyF = func() string {
			c, err := auth.NewClaims(
				auth.WithSubject("home"),
				auth.ExpiresIn(60*time.Minute),
			)
			if err != nil {
				slog.Error("Error creating claims for home view", "error", err)
				return ""
			}
			s, err := c.Sign(h.SigningKey)
			if err != nil {
				slog.Error("Error signing claims for home view", "error", err)
				return ""
			}
			return s
		}
	}
	tmpl = tmpl.Funcs(template.FuncMap{"AuthToken": keyF})
	homeSource, _ := home.ReadFile("pages/index.html")
	tmpl = template.Must(tmpl.Parse(string(homeSource)))
	return tmpl
}

func (h scrapeServer) homeHandler() http.HandlerFunc {
	tmpl := h.mustHomeTemplate()
	return func(w http.ResponseWriter, r *http.Request) {
		var buf bytes.Buffer
		if err := tmpl.Execute(&buf, nil); err != nil {
			http.Error(w, "Error rendering home page", http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		w.WriteHeader(http.StatusOK)
		w.Write(buf.Bytes())
	}
}

func (ss *scrapeServer) singleHandler() http.HandlerFunc {
	return Chain(ss.extract, ss.withAuthIfEnabled(MaxBytes(4096), parseSinglePayload())...)
}

func (ss *scrapeServer) singleHeadlessHandler() http.HandlerFunc {
	ms := ss.withAuthIfEnabled(MaxBytes(4096), parseSinglePayload())
	return Chain(extractWithFetcher(ss.headlessFetcher), ms...)
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
				slog.Error("Unexpected error fetching", "url", req.URL, "error", err)
				w.WriteHeader(http.StatusUnprocessableEntity)
			}
		}
		page.FetchMethod = resource.HeadlessChrome
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
			slog.Error("Unexpected error fetching", "url", req.URL, "error", err)
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

func (ss *scrapeServer) batchHandler() http.HandlerFunc {
	ms := ss.withAuthIfEnabled(MaxBytes(32768), DecodeJSONBody[BatchRequest]())
	return Chain(ss.batch, ms...)
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

func (ss *scrapeServer) deleteHandler() http.HandlerFunc {
	ms := ss.withAuthIfEnabled(MaxBytes(4096), parseSinglePayload())
	return Chain(ss.delete, ms...)
}

func (ss *scrapeServer) delete(w http.ResponseWriter, r *http.Request) {
	req, ok := r.Context().Value(payloadKey{}).(*singleURLRequest)
	if !ok {
		http.Error(w, "Can't process delete request, no input data", http.StatusInternalServerError)
		return
	}
	deleter, ok := ss.urlFetcher.(*scrape.StorageBackedFetcher)
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

func (ss *scrapeServer) feedHandler() http.HandlerFunc {
	ms := ss.withAuthIfEnabled(MaxBytes(4096), parseSinglePayload())
	return Chain(ss.feed, ms...)
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
