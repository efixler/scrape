package fetch

import (
	"context"
	"fmt"
	nurl "net/url"

	"github.com/efixler/scrape/resource"
)

type ErrHTTPError struct {
	StatusCode int
}

func (e ErrHTTPError) Error() string {
	return fmt.Sprintf("HTTP fetch error, code: %d", e.StatusCode)
}

func (e ErrHTTPError) String() string {
	return e.Error()
}

type Factory func() (URLData, error)

type URLData interface {
	Open(context.Context) error
	// Store(*StoredUrlData) (uint64, error)
	Fetch(*nurl.URL) (*resource.WebPage, error)
	Close() error
}
