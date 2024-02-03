package store

import (
	"errors"

	"github.com/efixler/scrape/fetch"
	"github.com/efixler/scrape/resource"
)

var (
	ErrorDatabaseNotFound = errors.New("database not found")
	ErrorResourceNotFound = errors.New("resource not found in data store")
	ErrorValueNotAllowed  = errors.New("value not allowed")
)

type Factory func() (URLDataStore, error)

// This interface is the contract for storing and retrieving WebPage resources.
// It adds a Store() method to the fetch.URLFetcher interface.
type URLDataStore interface {
	fetch.URLFetcher
	Store(*resource.WebPage) (uint64, error)
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
