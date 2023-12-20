package scrape

import (
	"context"
	"errors"
	"log"
	nurl "net/url"
	"sync"

	"github.com/efixler/scrape/fetch"
	"github.com/efixler/scrape/resource"
	"github.com/efixler/scrape/store"
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
	// check storage first
	item, err := f.storage.Fetch(url)
	if item != nil {
		return &item.Data, nil
	}
	if err != nil && !errors.Is(err, store.ErrorResourceNotFound) {
		return nil, err
	}

	// if we get here we're not cached
	resource, err := f.fetcher.Fetch(url)
	if err != nil {
		return nil, err
	}
	// store
	sd := &store.StoredUrlData{
		Data: *resource,
	}
	wg.Add(1)
	go func() {
		defer wg.Done()
		_, err = f.storage.Store(sd)
		if err != nil {
			log.Printf("Error storing %s: %s\n", url, err)
		}
	}()
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
