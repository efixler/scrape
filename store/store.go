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

const (
	DEFAULT_TTL = 24 * time.Hour * 30
)

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

// When we can close the gap between StoredUrlData and resource.WebPage, we can
// unite these interfaces maybe
// type URLDataStore interface {
// 	fetch.URLData
// 	Store(*StoredUrlData) (uint64, error)
// }

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

func (u *StoredUrlData) AssertTimes() {
	if u.Data.FetchTime == nil || u.Data.FetchTime.IsZero() {
		now := nowf().UTC().Truncate(time.Second)
		u.Data.FetchTime = &now
	}
	if u.TTL == nil {
		ttl := DEFAULT_TTL
		u.TTL = &ttl
	}
}
