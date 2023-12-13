package store

import (
	"errors"
	"hash/fnv"
	nurl "net/url"
	"time"

	"github.com/markusmobius/go-trafilatura"
)

const (
	DEFAULT_TTL = 24 * time.Hour * 30
)

var (
	ErrorDatabaseNotFound = errors.New("Database not found")
	nowf                  = time.Now
)

type StoredUrlData struct {
	Url         *nurl.URL
	Metadata    trafilatura.Metadata
	ContentText string
	TTL         *time.Duration
	FetchTime   *time.Time
}

type UrlDataStore interface {
	// Open(context.Context, string) (UrlDataStore, error)
	Store(StoredUrlData) (uint64, error)
	Fetch(*nurl.URL) (*StoredUrlData, error)
	Close() error
}

// todo: SQL supports int64 but not uint64 with high bit set
func GetKey(url *nurl.URL) uint32 {
	h := fnv.New32a()
	h.Write([]byte(url.String()))
	return h.Sum32()
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
