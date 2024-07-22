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
	ErrResourceNotFound = errors.New("resource not found in data store")
	ErrValueNotAllowed  = errors.New("value not allowed")
	ErrMappingNotFound  = errors.New("id mapping not found")
)

// This interface is the contract for storing and retrieving WebPage resources.
// It add Save() and Delete() methods to the fetch.URLFetcher interface.
type URLDataStore interface {
	fetch.URLFetcher
	Save(*resource.WebPage) (uint64, error)
	Delete(*nurl.URL) (bool, error)
}
