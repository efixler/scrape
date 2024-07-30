package trafilatura

import (
	"context"
	"errors"
	"log/slog"
	"mime"
	nurl "net/url"
	"strings"

	"github.com/efixler/scrape/fetch"
	"github.com/efixler/scrape/resource"
	_ "github.com/go-shiori/go-readability"
	_ "github.com/markusmobius/go-domdistiller"
	"github.com/markusmobius/go-trafilatura"
)

type TrafilaturaFetcher struct {
	client fetch.Client
}

func MustNew(client fetch.Client) fetch.URLFetcher {
	f, err := New(client)
	if err != nil {
		panic(err)
	}
	return f
}

func New(client fetch.Client) (*TrafilaturaFetcher, error) {
	var err error
	if client == nil {
		if client, err = fetch.NewClient(); err != nil {
			return nil, err
		}
	}
	fetcher := &TrafilaturaFetcher{
		client: client,
	}
	return fetcher, nil
}

func (f *TrafilaturaFetcher) Open(ctx context.Context) error {
	return nil
}

// Fetch a URL and return a WebPage resource.
// The web page will be fetched and parsed using the Trafilatura library.
// The returned resource will contain the metadata and content text.
// The request's StatusCode will be set to the HTTP status code returned.
// If there's an error fetching the page, in addition to the returned error,
// the *resource.WebPage will contain partial data pertaining to the request.
func (f *TrafilaturaFetcher) Fetch(url *nurl.URL) (*resource.WebPage, error) {
	var httpErr fetch.HttpError
	// FetchTime is inserted below
	rval := resource.NewWebPage(*url)
	resp, err := f.client.Get(url.String(), nil)
	if err != nil {
		// if we get an httpError back from doRequest, trust it
		if errors.As(err, &httpErr) {
			rval.StatusCode = httpErr.StatusCode
		} else if resp != nil {
			rval.StatusCode = resp.StatusCode
		}
		rval.Error = err
		return rval, err
	}

	defer resp.Body.Close()
	rval.StatusCode = resp.StatusCode
	if resp.StatusCode >= 400 || resp.StatusCode < 200 {
		// include the error in the resource, and return it.
		err = fetch.NewHTTPError(resp)
		rval.Error = err
		return rval, err
	}
	if ctype, _, err := mime.ParseMediaType(resp.Header.Get("Content-Type")); err != nil {
		slog.Warn("Error parsing Content-Type", "err", err, "url", url)
	} else {
		switch ctype {
		case "text/html":
		case "application/xhtml+xml": //todo: verify this
		case "text/plain":
		default: // trafilatura does grab some basic info from other content types,
			// but we don't want to try to parse them; the metadata can be wrong
			// and the data can be huge
			slog.Info("Unsupported Content-Type", "url", url, "ctype", ctype)
			err = fetch.NewUnsupportedContentTypeError(ctype)
			rval.Error = err
			return rval, err
		}
	}
	topts := trafilatura.Options{
		EnableFallback:     true,
		FallbackCandidates: &trafilatura.FallbackCandidates{},
		OriginalURL:        url,
		IncludeImages:      true,
	}
	result, err := trafilatura.Extract(resp.Body, topts)
	if err != nil {
		// there's an error that is thrown here that typically indicates
		// a JS-loaded page (that has no content at all, which isn't necessarily
		// true in all of these cases)
		// It's a plain error with the message:
		// "text and comments are not long enough: 0 0"
		return rval, err
	}
	f.applyExtractResult(result, rval)
	return rval, nil
}

func (f *TrafilaturaFetcher) applyExtractResult(
	tr *trafilatura.ExtractResult,
	r *resource.WebPage,
) {
	r.ContentText = tr.ContentText
	r.CanonicalURL, _ = nurl.Parse(tr.Metadata.URL)
	r.Title = tr.Metadata.Title
	r.Authors = make([]string, 0, 1)
	authors := strings.Split(tr.Metadata.Author, ";")
	for _, a := range authors {
		if trimmed := strings.TrimSpace(a); trimmed != "" {
			r.Authors = append(r.Authors, trimmed)
		}
	}
	r.Hostname = tr.Metadata.Hostname
	r.Description = tr.Metadata.Description
	r.Sitename = tr.Metadata.Sitename
	if !tr.Metadata.Date.IsZero() {
		r.Date = &tr.Metadata.Date
	}
	r.Categories = tr.Metadata.Categories
	r.Tags = tr.Metadata.Tags
	r.License = tr.Metadata.License
	r.Language = tr.Metadata.Language
	r.Image = tr.Metadata.Image
	r.PageType = tr.Metadata.PageType
	r.FetchMethod = f.client.Identifier()
}

func (f *TrafilaturaFetcher) Close() error {
	return nil
}
