package store

import (
	"errors"
	"fmt"
	nurl "net/url"

	"github.com/efixler/scrape/fetch"
	"github.com/efixler/scrape/resource"
)

var (
	ErrCantCreateDatabase = errors.New("can't create the database")
	ErrorDatabaseNotFound = errors.New("database not found")
	ErrorResourceNotFound = errors.New("resource not found in data store")
	ErrorValueNotAllowed  = errors.New("value not allowed")
	ErrMappingNotFound    = errors.New("id mapping not found")
)

type Factory func() (URLDataStore, error)

// This interface is the contract for storing and retrieving WebPage resources.
// It adds a Store() method to the fetch.URLFetcher interface.
// The fmt.Stringer interface is mainly to provide a way for the store to represent itself
// in log messages, e.g. a safe version of the DSN.
type URLDataStore interface {
	fetch.URLFetcher
	Save(*resource.WebPage) (uint64, error)
	Ping() error
	Delete(*nurl.URL) (bool, error)
	fmt.Stringer
}

// This interface adds create/clear/maintain methods to the URLDataStore interface.
// URLDataStores may support these methods to create, clear, and maintain the store.
type Maintainable interface {
	Create() error
	Clear() error
	Maintain() error
}

// This interface is to expose a method to supply data to healthchecks.
type Observable interface {
	Stats() (any, error)
}
