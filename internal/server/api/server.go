// Implements API method handlers for scrape-server.
package api

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	nurl "net/url"

	"github.com/efixler/scrape/database"
	"github.com/efixler/scrape/fetch"
	"github.com/efixler/scrape/fetch/feed"
	"github.com/efixler/scrape/internal"
	"github.com/efixler/scrape/internal/auth"
	"github.com/efixler/scrape/internal/server/middleware"
	"github.com/efixler/scrape/internal/settings"
	"github.com/efixler/scrape/resource"
	"github.com/efixler/webutil/jsonarray"
)

const (
	TokenCookieName = "token"
)

func WithURLFetcher(f fetch.URLFetcher) option {
	return func(s *Server) error {
		if f == nil {
			return errors.New("nil fetcher provided")
		}
		s.urlFetcher = f
		return nil
	}
}

func WithHeadlessIf(hf fetch.URLFetcher) option {
	return func(s *Server) error {
		if hf == nil {
			return nil
		}
		s.headlessFetcher = hf
		return nil
	}
}

func WithFeedFetcher(ff fetch.FeedFetcher) option {
	return func(s *Server) error {
		if ff == nil {
			return errors.New("nil feed fetcher provided")
		}
		s.feedFetcher = ff
		return nil
	}
}

func WithSettingsFrom(db *database.DBHandle) option {
	return func(s *Server) error {
		if db == nil {
			return errors.New("nil database handle provided")
		}
		s.settingsStorage = settings.NewDomainSettingsStorage(db)
		return nil
	}
}

// Sets the HMAC key and enables JWT authorization if the key is non-empty.
func WithAuthorizationIf(key auth.HMACBase64Key) option {
	return func(s *Server) error {
		if len(key) > 0 {
			s.signingKey = key
		}
		return nil
	}
}

type option func(*Server) error

func MustAPIServer(ctx context.Context, opts ...option) *Server {
	ss, err := NewAPIServer(ctx, opts...)
	if err != nil {
		panic(err)
	}
	return ss
}

func NewAPIServer(ctx context.Context, opts ...option) (*Server, error) {
	ss := &Server{ctx: ctx}
	for _, opt := range opts {
		err := opt(ss)
		if err != nil {
			return nil, err
		}
	}
	if ss.urlFetcher == nil {
		return nil, errors.New("no URL fetcher provided")
	}
	if ss.feedFetcher == nil {
		ss.feedFetcher = feed.MustFeedFetcher()
	}
	return ss, nil
}

// The API server is the main entry point for the scrape-server API.
type Server struct {
	ctx             context.Context
	urlFetcher      fetch.URLFetcher
	headlessFetcher fetch.URLFetcher
	feedFetcher     fetch.FeedFetcher
	signingKey      auth.HMACBase64Key
	settingsStorage settings.DomainSettingsStore
}

func (ss Server) SigningKey() auth.HMACBase64Key {
	return ss.signingKey
}

func (ss Server) AuthEnabled() bool {
	return len(ss.signingKey) > 0
}

// Prepend the authorization checker to the list of passed middleware if authorization is enabled.
func (ss Server) withAuthIfEnabled(ms ...middleware.Step) []middleware.Step {
	if len(ss.signingKey) > 0 {
		ms = append([]middleware.Step{
			auth.JWTAuthzMiddleware(ss.signingKey, auth.WithCookie(TokenCookieName))},
			ms...,
		)
	}
	return ms
}

func (ss *Server) Extract() http.HandlerFunc {
	return middleware.Chain(
		ss.extract,
		ss.withAuthIfEnabled(middleware.MaxBytes(4096), parseSinglePayload())...,
	)
}

func (ss *Server) ExtractHeadless() http.HandlerFunc {
	ms := ss.withAuthIfEnabled(middleware.MaxBytes(4096), parseSinglePayload())
	return middleware.Chain(extractWithFetcher(ss.headlessFetcher), ms...)
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
		req, _ := r.Context().Value(payloadKey{}).(*SingleURLRequest)
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
		page.FetchMethod = resource.HeadlessChromium
		encoder := json.NewEncoder(w)
		encoder.SetEscapeHTML(false)
		if req.PrettyPrint {
			encoder.SetIndent("", "  ")
		}
		encoder.Encode(page)
	}
}

func (h *Server) extract(w http.ResponseWriter, r *http.Request) {
	req, ok := r.Context().Value(payloadKey{}).(*SingleURLRequest)
	if !ok {
		http.Error(w, "Can't process extract request, no input data", http.StatusInternalServerError)
		return
	}
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

func (ss *Server) Batch() http.HandlerFunc {
	ms := ss.withAuthIfEnabled(
		middleware.MaxBytes(32768),
		middleware.DecodeJSONBody[BatchRequest](payloadKey{}),
	)
	return middleware.Chain(ss.batch, ms...)
}

func (h *Server) batch(w http.ResponseWriter, r *http.Request) {
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

func (ss *Server) Delete() http.HandlerFunc {
	ms := ss.withAuthIfEnabled(middleware.MaxBytes(4096), parseSinglePayload())
	return middleware.Chain(ss.delete, ms...)
}

func (ss *Server) delete(w http.ResponseWriter, r *http.Request) {
	req, ok := r.Context().Value(payloadKey{}).(*SingleURLRequest)
	if !ok {
		http.Error(w, "Can't process delete request, no input data", http.StatusInternalServerError)
		return
	}
	deleter, ok := ss.urlFetcher.(*internal.StorageBackedFetcher)
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

func (h *Server) synchronousBatch(urls []string, encoder *jsonarray.Encoder[*resource.WebPage]) {
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

func (ss *Server) Feed() http.HandlerFunc {
	ms := ss.withAuthIfEnabled(middleware.MaxBytes(4096), parseSinglePayload())
	return middleware.Chain(ss.feed, ms...)
}

func (h *Server) feed(w http.ResponseWriter, r *http.Request) {
	req, ok := r.Context().Value(payloadKey{}).(*SingleURLRequest)
	if !ok {
		http.Error(w, "Can't process extract request, no input data", http.StatusInternalServerError)
		return
	}
	resource, err := h.feedFetcher.FetchContext(r.Context(), req.URL)
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
