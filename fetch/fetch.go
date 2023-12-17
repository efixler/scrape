package fetch

import (
	"context"
	nurl "net/url"

	"github.com/efixler/scrape/resource"
)

type Factory func() (URLData, error)

type URLData interface {
	Open(context.Context) error
	// Store(*StoredUrlData) (uint64, error)
	Fetch(*nurl.URL) (*resource.WebPage, error)
	Close() error
}
