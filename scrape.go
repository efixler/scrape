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

	"log/slog"

	"github.com/efixler/scrape/fetch"
	"github.com/efixler/scrape/resource"
	"github.com/efixler/scrape/store"
)

var (
	wg sync.WaitGroup
)

type StorageBackedFetcher struct {
	Fetcher fetch.URLData
	Storage store.URLDataStore
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
		Fetcher: fetcher,
		Storage: storage,
	}, nil
}

// The context passed to Open() will be passed on to child components
// so that they can hook into the context directly, specifically to
// close and release resources on cancellation.
func (f StorageBackedFetcher) Open(ctx context.Context) error {
	err := f.Fetcher.Open(ctx)
	if err != nil {
		return err
	}
	err = f.Storage.Open(ctx)
	if err != nil {
		return err
	}
	// We actually shouldn't need this, since the child components will hook into the context
	// directly.
	context.AfterFunc(ctx, func() {
		f.Close()
	})
	return nil
}

func (f StorageBackedFetcher) Fetch(url *nurl.URL) (*resource.WebPage, error) {
	// Treat this as the entry point for the url and apply cleaning here.
	// Re-evaluate this -- right now this is the entry point for all URLs,
	// but it might not always be. We also might want storage to use the cleaned URLs,
	// but always fetch the url precisely as requested.
	originalURL := url.String()
	// Now clean the URL
	url = resource.CleanURL(url)
	// Now fetch the item from storage
	item, err := f.Storage.Fetch(url)
	if err != nil && !errors.Is(err, store.ErrorResourceNotFound) {
		return nil, err
	}
	var resource *resource.WebPage
	if item != nil {
		resource = &item.Data
	}
	// TODO: Check that we don't store OriginalURL with the resource
	defer func() { resource.OriginalURL = originalURL }()
	if resource == nil {
		resource, err = f.Fetcher.Fetch(url)
		if err != nil {
			return resource, err
		}
		sd := &store.StoredUrlData{
			Data: *resource,
		}
		wg.Add(1)
		go func() {
			defer wg.Done()
			key, err := f.Storage.Store(sd)
			if err != nil {
				slog.Error("Error storing %s: %s\n", "url", url, "key", key, "error", err)
			}
		}()
	}
	return resource, nil
}

// Close() will be invoked when the context sent to Open() is done
// If that context doesn't get cancelled, Close() must be called to
// release resources.
func (f *StorageBackedFetcher) Close() error {
	if f.closed {
		return nil
	}
	defer func() {
		f.closed = true
	}()
	f.Fetcher.Close()
	f.Storage.Close()
	return nil
}
