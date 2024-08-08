/*
This package, and its subpackages, generally provide support for `scrape`,
`scrape-server`, the other tools in the top-level cmd/ directory.
*/
package internal

import (
	"context"
	"errors"
	nurl "net/url"
	"sync"

	"log/slog"

	"github.com/efixler/scrape/database"
	"github.com/efixler/scrape/fetch"
	"github.com/efixler/scrape/internal/storage"
	"github.com/efixler/scrape/resource"
)

type URLStore interface {
	fetch.URLFetcher
	Database() *database.DBHandle
	Save(*resource.WebPage) (uint64, error)
	Delete(*nurl.URL) (bool, error)
}

// StorageBackedFetcher returns URLs from a storage backend, and fetches them if they are not found.
type StorageBackedFetcher struct {
	Fetcher fetch.URLFetcher
	Storage URLStore
	saving  *sync.WaitGroup
	closed  bool
}

// NewStorageBackedFetcher returns a new StorageBackedFetcher that uses the given fetcher and storage.
// It will add a CloseListener to the storage to wait for pending saves.
func NewStorageBackedFetcher(
	fetcher fetch.URLFetcher,
	storage URLStore,
) *StorageBackedFetcher {
	s := &StorageBackedFetcher{
		Fetcher: fetcher,
		Storage: storage,
		saving:  new(sync.WaitGroup),
	}
	s.Storage.Database().AddCloseListener(func() {
		s.Wait()
	})
	return s
}

// WithAlternateURLFetcher returns a new StorageBatchedFetcher using the same storage but a different url fetcher.
// This is to support headless fetching, where we want to use a different underlying http retrieval client
// but the same storage.
func (f *StorageBackedFetcher) WithAlternateURLFetcher(ctx context.Context, uf fetch.URLFetcher) (*StorageBackedFetcher, error) {
	if f.closed {
		return nil, errors.New("StorageBackedFetcher is closed")
	}
	clone := &StorageBackedFetcher{
		Fetcher: uf,
		Storage: f.Storage,
		saving:  f.saving,
	}
	// Don't patch in a function to close the context here, because we only really need this to close the DB, which is already
	// hooked by the parent. We also share the parent's WaitGroup for async saves for this reason.
	return clone, nil
}

func (f *StorageBackedFetcher) Fetch(url *nurl.URL) (*resource.WebPage, error) {
	// Treat this as the entry point for the url and apply cleaning here.
	originalURL := url.String()
	url = resource.CleanURL(url)
	// Now fetch the item from storage
	res, err := f.Storage.Fetch(url)
	if err != nil && !errors.Is(err, storage.ErrResourceNotFound) {
		return nil, err
	}
	defer func() { res.OriginalURL = originalURL }()
	if res == nil {
		res, err = f.Fetcher.Fetch(url)
		// never store a resource with an error, but do return a partial resource
		if err != nil {
			return res, err
		}
		f.saving.Add(1)
		go func() {
			defer f.saving.Done()
			key, err := f.Storage.Save(res)
			if err != nil {
				slog.Error("Error storing %s: %s\n", "url", url, "key", key, "error", err)
			}
		}()
	}
	return res, nil
}

// Batch fetches a batch of URLs, returning a channel of WebPages. The channel is not guaranteed to return
// URLs in the order they were requested.
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

// Wait() will block on pending saves.
func (f *StorageBackedFetcher) Wait() error {
	if f.closed {
		return nil
	}
	defer func() {
		f.closed = true
	}()
	f.saving.Wait()
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
				if _, err := f.Storage.Save(res); err != nil {
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
		} else if errors.Is(err, storage.ErrResourceNotFound) {
			notFoundChan <- fetchMsg{cleanedURL: url, originalURL: originalURL}
		} else { // this is really an error
			slog.Error("Error fetching url in Batch", "url", url, "error", err)
		}
	}
}

func (f StorageBackedFetcher) Delete(url *nurl.URL) (bool, error) {
	return f.Storage.Delete(url)
}
