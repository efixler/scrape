package store

import (
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
	nowf                  = time.Now
)

type StoredUrlData struct {
	Data      resource.WebPage
	TTL       *time.Duration
	FetchTime *time.Time
}

type UrlDataStore interface {
	// Open(context.Context, string) (UrlDataStore, error)
	Store(StoredUrlData) (uint64, error)
	Fetch(*nurl.URL) (*StoredUrlData, error)
	Close() error
}

func (u *StoredUrlData) AssertTimes() {
	if u.FetchTime == nil || u.FetchTime.IsZero() {
		now := nowf()
		u.FetchTime = &now
	}
	if u.TTL == nil {
		ttl := DEFAULT_TTL
		u.TTL = &ttl
	}
}
