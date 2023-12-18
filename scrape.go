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
	err := f.storage.Open(ctx)
	if err != nil {
		return err
	}
	err = f.fetcher.Open(ctx)
	if err != nil {
		return err
	}
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

// You must exolicitly call Close() on this object to ensure that
// all resources are released.
func (f StorageBackedFetcher) Close() error {
	wg.Add(1)
	go func() {
		defer wg.Done()
		err := f.fetcher.Close()
		if err != nil {
			log.Printf("Error closing fetcher: %s", err)
		}
	}()
	wg.Add(1)
	go func() {
		defer wg.Done()
		err := f.storage.Close()
		if err != nil {
			log.Printf("Error closing storage: %s", err)
		}
	}()
	wg.Wait()
	return nil
}
