package feed

import (
	"context"
	"net/http"
	nurl "net/url"
	"time"

	"github.com/efixler/scrape/fetch"
	"github.com/efixler/scrape/resource"
	"github.com/mmcdole/gofeed"
)

const (
	DefaultTimeout = 30 * time.Second
)

var (
	DefaultOptions = Options{
		Timeout:   DefaultTimeout,
		UserAgent: fetch.DefaultUserAgent,
	}
)

type Options struct {
	UserAgent string
	Timeout   time.Duration
	Client    *http.Client
}

type FeedFetcher struct {
	parser  *gofeed.Parser
	ctx     context.Context
	timeout time.Duration
}

func NewFeedFetcher(options Options) *FeedFetcher {
	parser := gofeed.NewParser()
	if options.UserAgent != "" {
		parser.UserAgent = options.UserAgent
	}
	if options.Client != nil {
		parser.Client = options.Client
	}
	if options.Timeout == 0 {
		options.Timeout = DefaultTimeout
	}
	return &FeedFetcher{
		parser:  parser,
		timeout: options.Timeout,
	}
}

func (f *FeedFetcher) Open(ctx context.Context) error {
	f.ctx = ctx
	context.AfterFunc(ctx, func() {
		f.Close()
	})
	return nil
}

func (f *FeedFetcher) Fetch(url *nurl.URL) (*resource.Feed, error) {
	ctx, cancel := context.WithTimeout(f.ctx, f.timeout)
	defer cancel()
	feed, err := f.parser.ParseURLWithContext(url.String(), ctx)
	if err != nil {
		return nil, err
	}
	return &resource.Feed{
		Feed: *feed,
	}, nil
}

func (f *FeedFetcher) Close() error {
	return nil
}
