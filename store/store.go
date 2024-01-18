package store

import (
	"context"
	"errors"
	"fmt"
	nurl "net/url"
	"time"

	"github.com/efixler/scrape/resource"
)

type DatabaseOptions fmt.Stringer

var (
	ErrorDatabaseNotFound = errors.New("database not found")
	ErrorResourceNotFound = errors.New("resource not found in data store")
	nowf                  = time.Now
)

type Factory func() (URLDataStore, error)

type StoredUrlData struct {
	Data resource.WebPage //todo promote this to default embedding
	TTL  *time.Duration
}

type URLDataStore interface {
	Open(context.Context) error
	Store(*StoredUrlData) (uint64, error)
	Fetch(*nurl.URL) (*StoredUrlData, error)
	Close() error
}

type Maintainable interface {
	Create() error
	Clear() error
	Maintain() error
}

type Observable interface {
	Stats() (any, error)
}

func (u *StoredUrlData) AssertTimes() {
	if u.Data.FetchTime == nil || u.Data.FetchTime.IsZero() {
		now := nowf().UTC().Truncate(time.Second)
		u.Data.FetchTime = &now
	}
	if u.TTL == nil {
		ttl := resource.DefaultTTL
		u.TTL = &ttl
	}
}
