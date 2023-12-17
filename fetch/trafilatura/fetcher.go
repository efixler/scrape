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

var trafilaturaFallback = &trafilatura.FallbackConfig{}

type TrafilaturaFetcher struct {
	httpClient *http.Client
	ctx        context.Context
}

func Factory() func() (fetch.URLData, error) {
	return func() (fetch.URLData, error) {
		return NewTrafilaturaFetcher(), nil
	}
}

func NewTrafilaturaFetcher() *TrafilaturaFetcher {
	return &TrafilaturaFetcher{
		httpClient: &http.Client{Timeout: 30 * time.Second},
	}
}

func (f *TrafilaturaFetcher) Open(ctx context.Context) error {
	f.ctx = ctx
	return nil
}

func (f *TrafilaturaFetcher) Fetch(url *nurl.URL) (*resource.WebPage, error) {
	resp, err := f.httpClient.Get(url.String())
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	topts := trafilatura.Options{
		FallbackCandidates: trafilaturaFallback,
		OriginalURL:        url,
		IncludeImages:      true,
	}
	result, err := trafilatura.Extract(resp.Body, topts)
	if err != nil {
		return nil, err
	}
	fetchTime := time.Now().UTC()
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
