// Implements a fetcher for RSS/Atom feeds using the gofeed library.
package feed

import (
	"context"
	"errors"
	"fmt"
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

type option func(*config) error

func WithUserAgent(ua string) option {
	return func(c *config) error {
		if ua == "" {
			return errors.New("user agent must not be empty")
		}
		c.UserAgent = ua
		return nil
	}
}

func WithTimeout(t time.Duration) option {
	return func(c *config) error {
		if t <= 0 {
			return errors.New("timeout must be positive")
		}
		c.Timeout = t
		return nil
	}
}

func WithClient(client *http.Client) option {
	return func(c *config) error {
		c.Client = client
		return nil
	}
}

var (
	DefaultConfig = config{
		Timeout:   DefaultTimeout,
		UserAgent: fetch.DefaultUserAgent,
	}
)

type config struct {
	UserAgent string
	Timeout   time.Duration
	Client    *http.Client
}

type FeedFetcher struct {
	parser  *gofeed.Parser
	timeout time.Duration
}

func MustFeedFetcher(options ...option) *FeedFetcher {
	f, err := NewFeedFetcher(options...)
	if err != nil {
		panic(err)
	}
	return f
}

func NewFeedFetcher(options ...option) (*FeedFetcher, error) {
	config := DefaultConfig
	for _, opt := range options {
		if err := opt(&config); err != nil {
			return nil, err
		}
	}
	parser := gofeed.NewParser()
	parser.UserAgent = config.UserAgent

	if config.Client != nil {
		parser.Client = config.Client
	}
	return &FeedFetcher{
		parser:  parser,
		timeout: config.Timeout,
	}, nil
}

func (f *FeedFetcher) Fetch(url *nurl.URL) (*resource.Feed, error) {
	ctx, cancel := context.WithTimeout(context.Background(), f.timeout)
	defer cancel()
	return f.FetchContext(ctx, url)
}

func (f *FeedFetcher) FetchContext(ctx context.Context, url *nurl.URL) (*resource.Feed, error) {
	feed, err := f.parser.ParseURLWithContext(url.String(), ctx)
	if err != nil {
		if errors.Is(err, context.DeadlineExceeded) {
			return nil, fetch.HttpError{
				StatusCode: http.StatusGatewayTimeout,
				Status:     http.StatusText(http.StatusGatewayTimeout),
				Message:    fmt.Sprintf("%s did not reply within %v seconds", url.String(), f.timeout.Seconds()),
			}
		}
		return nil, err
	}
	return &resource.Feed{
		Feed:         *feed,
		RequestedURL: url.String(),
	}, nil
}
