// Package store defines interfaces for types that handle
// storage of web page metadata, along with definitions of
// a few common errors.
package store

import (
	"errors"
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
	Ping() error // remove me
	Delete(*nurl.URL) (bool, error)
}
