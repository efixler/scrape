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
)

type TrafilaturaFetcher struct {
	httpClient *http.Client
	ctx        context.Context
	userAgent  string
}

func Factory() func() (fetch.URLData, error) {
	return func() (fetch.URLData, error) {
		// todo: make this configurable
		return NewTrafilaturaFetcher(DefaultUserAgent, nil), nil
	}
}

func NewTrafilaturaFetcher(userAgent string, transport http.RoundTripper) *TrafilaturaFetcher {
	client := &http.Client{Timeout: 30 * time.Second}
	if transport != nil {
		client.Transport = transport
	}
	return &TrafilaturaFetcher{
		httpClient: client,
		userAgent:  userAgent,
	}
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
		return nil, fetch.ErrHTTPError{StatusCode: resp.StatusCode}
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
