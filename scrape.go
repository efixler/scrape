/*
Package scrape provides a simple interface for fetching and storing web pages'
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
	Fetcher fetch.URLFetcher
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
	context.AfterFunc(ctx, func() {
		f.Close()
	})
	return nil
}

func (f StorageBackedFetcher) Fetch(url *nurl.URL) (*resource.WebPage, error) {
	// Treat this as the entry point for the url and apply cleaning here.
	originalURL := url.String()
	url = resource.CleanURL(url)
	// Now fetch the item from storage
	res, err := f.Storage.Fetch(url)
	if err != nil && !errors.Is(err, store.ErrorResourceNotFound) {
		return nil, err
	}
	defer func() { res.OriginalURL = originalURL }()
	if res == nil {
		res, err = f.Fetcher.Fetch(url)
		// never store a resource with an error, but do return a partial resource
		if err != nil {
			return res, err
		}
		wg.Add(1)
		go func() {
			defer wg.Done()
			key, err := f.Storage.Store(res)
			if err != nil {
				slog.Error("Error storing %s: %s\n", "url", url, "key", key, "error", err)
			}
		}()
	}
	return res, nil
}

func (f StorageBackedFetcher) Batch(urls []string, options fetch.BatchOptions) <-chan *resource.WebPage {
	rchan := make(chan *resource.WebPage, len(urls))
	unstoredChan := make(chan fetchMsg)

	// This lets us wait on the goroutines to finish so we we can close the returned channel
	var wg sync.WaitGroup
	wg.Add(2)
	go func() {
		wg.Wait()
		close(rchan)
	}()

	// start the go func to fetch the urls if they aren't stored
	go func() {
		defer wg.Done()
		f.fetchUnstored(unstoredChan, rchan)
	}()

	// start the go func that loads from the DB
	go func() {
		defer wg.Done()
		f.loadBatch(urls, rchan, unstoredChan)
	}()
	return rchan
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

type fetchMsg struct {
	cleanedURL  *nurl.URL
	originalURL string
}

// TODO: Apply rate limiting here
func (f *StorageBackedFetcher) fetchUnstored(inchan <-chan fetchMsg, outchan chan<- *resource.WebPage) {
	for msg := range inchan {
		res, err := f.Fetcher.Fetch(msg.cleanedURL)
		rcopy := *res
		rcopy.OriginalURL = msg.originalURL
		rcopy.Error = err
		outchan <- &rcopy
		if err == nil {
			go func() {
				if _, err := f.Storage.Store(res); err != nil {
					slog.Error("Error storing %s: %s\n", "url", res.RequestedURL, "error", err)
				}
			}()
		}
	}
}

func (f *StorageBackedFetcher) loadBatch(
	urls []string,
	foundChan chan<- *resource.WebPage,
	notFoundChan chan<- fetchMsg) {
	defer close(notFoundChan)
	var (
		parsedURL *nurl.URL
		err       error
	)
	for _, originalURL := range urls {
		if parsedURL, err = nurl.Parse(originalURL); err != nil {
			foundChan <- &resource.WebPage{
				OriginalURL: originalURL,
				Error:       err,
			}
			continue
		}
		url := resource.CleanURL(parsedURL)
		if res, err := f.Storage.Fetch(url); err == nil {
			res.OriginalURL = originalURL
			foundChan <- res
		} else if errors.Is(err, store.ErrorResourceNotFound) {
			notFoundChan <- fetchMsg{cleanedURL: url, originalURL: originalURL}
		} else { // this is really an error
			slog.Error("Error fetching url in Batch", "url", url, "error", err)
		}
	}
}
