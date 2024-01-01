package trafilatura

import (
	"context"
	"net/http"
	nurl "net/url"
	"time"

	"github.com/efixler/scrape/fetch"
	"github.com/efixler/scrape/resource"
	_ "github.com/go-shiori/go-readability"
	_ "github.com/markusmobius/go-domdistiller"
	"github.com/markusmobius/go-trafilatura"
)

const (
	DefaultUserAgent = "Mozilla/5.0 (X11; Linux x86_64; rv:88.0) Gecko/20100101 Firefox/88.0"
)

var (
	trafilaturaFallback = &trafilatura.FallbackConfig{}
	DefaultOptions      = &Options{
		FallbackConfig: trafilaturaFallback,
		HttpClient:     &http.Client{Timeout: 30 * time.Second},
		UserAgent:      DefaultUserAgent,
		Transport:      nil,
	}
)

type Options struct {
	FallbackConfig *trafilatura.FallbackConfig
	HttpClient     *http.Client
	UserAgent      string
	Transport      http.RoundTripper
}

// func Defaults() Options {
// 	return *defaultOptions
// }

type TrafilaturaFetcher struct {
	httpClient *http.Client
	ctx        context.Context
	userAgent  string
}

// Factory function for new fetcher.
func Factory(options Options) func() (fetch.URLData, error) {
	// Implemented as a factory for some concurrency possbilities but
	// we might not need this now (or at all)
	return func() (fetch.URLData, error) {
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
		return nil, err
	}
	return resp, nil
}

func (f *TrafilaturaFetcher) Fetch(url *nurl.URL) (*resource.WebPage, error) {
	resp, err := f.doRequest(url.String())
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 400 {
		return nil, fetch.NewErrHTTPError(resp.StatusCode, resp.Body)
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
		return nil, err
	}
	fetchTime := time.Now().UTC().Truncate(time.Second)
	resource := &resource.WebPage{
		Metadata:     result.Metadata,
		ContentText:  result.ContentText,
		RequestedURL: url,
		FetchTime:    &fetchTime,
	}
	return resource, nil
}

func (f *TrafilaturaFetcher) Close() error {
	return nil
}
