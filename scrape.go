/*
Package scrape provides a simple interface for fetching and storing web pages
metadata and text content. The `scrape` and `scrape-server` commands provide
a command-line interface and a REST API, respectively.
*/
package scrape

import (
	"context"
	"errors"
	nurl "net/url"
	"sync"

	"github.com/efixler/scrape/fetch"
	"github.com/efixler/scrape/resource"
	"github.com/efixler/scrape/store"
	"golang.org/x/exp/slog"
)

var (
	wg sync.WaitGroup
)

type StorageBackedFetcher struct {
	fetcher fetch.URLData
	storage store.URLDataStore
	closed  bool
}

func NewStorageBackedFetcher(
	fetcherFactory fetch.Factory,
	storageFactory store.Factory,
) (*StorageBackedFetcher, error) {
	storage, err := storageFactory()
	if err != nil {
		return nil, err
	}
	fetcher, err := fetcherFactory()
	if err != nil {
		return nil, err
	}
	return &StorageBackedFetcher{
		fetcher: fetcher,
		storage: storage,
	}, nil
}

// We will need the ctx here at some point (and will need to change to a reference pointer)
func (f StorageBackedFetcher) Open(ctx context.Context) error {
	err := f.fetcher.Open(ctx)
	if err != nil {
		return err
	}
	err = f.storage.Open(ctx)
	if err != nil {
		return err
	}
	context.AfterFunc(ctx, func() {
		f.Close()
	})
	return nil
}

func (f StorageBackedFetcher) Fetch(url *nurl.URL) (*resource.WebPage, error) {
	originalURL := url.String()
	// check storage first
	item, err := f.storage.Fetch(url)
	if err != nil && !errors.Is(err, store.ErrorResourceNotFound) {
		return nil, err
	}
	var resource *resource.WebPage
	if item != nil {
		resource = &item.Data
	}
	if resource == nil {
		resource, err = f.fetcher.Fetch(url)
		if err != nil {
			return nil, err
		}
		sd := &store.StoredUrlData{
			Data: *resource,
		}
		wg.Add(1)
		go func() {
			defer wg.Done()
			_, err = f.storage.Store(sd)
			if err != nil {
				slog.Error("Error storing %s: %s\n", "url", url, "error", err)
			}
		}()
	}
	resource.OriginalURL = originalURL
	return resource, nil
}

// Close() will be invoked when the context sent to Open() is done
// If that context doesn't get cancelled, Close() must be called to
// release resources
func (f *StorageBackedFetcher) Close() error {
	if f.closed {
		return nil
	}
	defer func() {
		f.closed = true
	}()
	f.fetcher.Close()
	f.storage.Close()

	return nil
}
