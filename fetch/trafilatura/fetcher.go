package trafilatura

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"mime"
	"net"
	"net/http"
	nurl "net/url"
	"time"

	"github.com/efixler/scrape/fetch"
	"github.com/efixler/scrape/resource"
	_ "github.com/go-shiori/go-readability"
	_ "github.com/markusmobius/go-domdistiller"
	"github.com/markusmobius/go-trafilatura"
)

var (
	DefaultUserAgent    = "Mozilla/5.0 (X11; Linux x86_64; rv:88.0) Gecko/20100101 Firefox/88.0"
	DefaultTimeout      = 30 * time.Second
	trafilaturaFallback = &trafilatura.FallbackConfig{}
	DefaultOptions      = &Options{
		FallbackConfig: trafilaturaFallback,
		HttpClient:     &http.Client{},
		Timeout:        DefaultTimeout,
		Transport:      nil,
		UserAgent:      fetch.DefaultUserAgent,
	}
)

type Options struct {
	FallbackConfig *trafilatura.FallbackConfig
	HttpClient     *http.Client
	UserAgent      string
	Transport      http.RoundTripper
	Timeout        time.Duration
}

type TrafilaturaFetcher struct {
	httpClient *http.Client
	ctx        context.Context
	userAgent  string
}

// Factory function for new fetcher.
func Factory(options Options) func() (fetch.URLFetcher, error) {
	// Implemented as a factory for some concurrency possbilities but
	// we might not need this now (or at all)
	return func() (fetch.URLFetcher, error) {
		return NewTrafilaturaFetcher(options), nil
	}
}

func NewTrafilaturaFetcher(options Options) *TrafilaturaFetcher {

	if options.FallbackConfig == nil {
		options.FallbackConfig = DefaultOptions.FallbackConfig
	}
	if options.HttpClient == nil {
		options.HttpClient = DefaultOptions.HttpClient
	}
	if options.UserAgent == "" {
		options.UserAgent = DefaultOptions.UserAgent
	}
	fetcher := &TrafilaturaFetcher{
		httpClient: options.HttpClient,
		userAgent:  options.UserAgent,
	}
	// if the httpClient has a timeout set, assume it's intentional and use it
	// but make sure we never use a timeout of 0
	if fetcher.httpClient.Timeout == 0 {
		fetcher.httpClient.Timeout = options.Timeout
	}
	if fetcher.httpClient.Timeout == 0 {
		fetcher.httpClient.Timeout = DefaultTimeout
	}
	if options.Transport != nil {
		fetcher.httpClient.Transport = options.Transport
	}
	return fetcher
}

func (f *TrafilaturaFetcher) Open(ctx context.Context) error {
	f.ctx = ctx
	return nil
}

func (f *TrafilaturaFetcher) doRequest(url string) (*http.Response, error) {
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", f.userAgent)
	resp, err := f.httpClient.Do(req)
	if err != nil {
		if netErr, ok := err.(net.Error); ok && netErr.Timeout() {
			return nil, fetch.HttpError{
				StatusCode: http.StatusGatewayTimeout,
				Status:     http.StatusText(http.StatusGatewayTimeout),
				Message: fmt.Sprintf(
					"%s did not reply within %v seconds",
					url,
					f.httpClient.Timeout.Seconds(),
				),
			}
		}
		return resp, err
	}
	return resp, err
}

// Fetch a URL and return a WebPage resource.
// The web page will be fetched and parsed using the Trafilatura library.
// The returned resource will contain the metadata and content text.
// The request's StatusCode will be set to the HTTP status code returned.
// If there's an error fetching the page, in addition to the returned error,
// the *resource.WebPage will contain partial data pertaining to the request.
func (f *TrafilaturaFetcher) Fetch(url *nurl.URL) (*resource.WebPage, error) {
	fetchTime := time.Now().UTC().Truncate(time.Second)
	var httpErr fetch.HttpError
	rval := &resource.WebPage{
		RequestedURL: url,
		FetchTime:    &fetchTime,
	}
	resp, err := f.doRequest(url.String())
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
		FallbackCandidates: trafilaturaFallback,
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
	rval.Metadata = result.Metadata
	rval.ContentText = result.ContentText
	return rval, nil
}

func (f *TrafilaturaFetcher) Close() error {
	return nil
}
