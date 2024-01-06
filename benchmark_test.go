//go:build benchmark

package scrape

// This file is associated with the tag "benchmark" to keep it from running
// during normal tests.  It can take a while, and some of the tests call out,
// which can lead to IP blocks. Needs to be revisited.

import (
	"context"
	"fmt"
	"net/http"
	nurl "net/url"
	"os"
	"strings"
	"testing"

	"github.com/efixler/scrape/fetch/trafilatura"
	"github.com/efixler/scrape/store"
	"github.com/efixler/scrape/store/sqlite"
)

const (
	HTMLDir = "./internal/testdata/scraped/html"
	DBFile  = "./internal/testdata/scrape.db"
	CsvFile = "./internal/testdata/global_urls.csv"
	MaxURLs = 1000
)

func loadUrls() ([]*nurl.URL, error) {
	files, err := os.ReadDir(HTMLDir)
	if err != nil {
		return nil, err
	}
	var urls = make([]*nurl.URL, 0, MaxURLs)
	for _, file := range files {
		if file.IsDir() {
			continue
		}
		fname := file.Name()
		if !strings.HasSuffix(fname, ".html") {
			continue
		}
		url, err := nurl.Parse("file://html/" + fname)
		if err != nil {
			return nil, err
		}
		urls = append(urls, url)
		if len(urls) >= MaxURLs {
			break
		}
	}
	fmt.Printf("Loaded %d URLs\n", len(urls))
	return urls, nil
}

func makeFetcher(dbPath string, ctx context.Context) (*StorageBackedFetcher, error) {
	t := &http.Transport{}
	t.RegisterProtocol("file", http.NewFileTransport(http.Dir(HTMLDir)))
	topts := *trafilatura.DefaultOptions
	topts.Transport = t

	fetcher, err := NewStorageBackedFetcher(
		trafilatura.Factory(topts),
		sqlite.Factory(dbPath),
	)
	if err != nil {
		return nil, err
	}
	err = fetcher.Open(ctx)
	if err != nil {
		return nil, err
	}
	return fetcher, nil
}

func BenchmarkWarmupSqliteFileDB(b *testing.B) {
	b.Skip()
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	fetcher, err := makeFetcher(DBFile, ctx)
	if err != nil {
		b.Fatal(err)
	}
	urls, err := loadUrls()
	if err != nil {
		b.Fatal(err)
	}
	dbm, _ := fetcher.Storage.(store.Maintainable)
	for i := 0; i < b.N; i++ {
		for _, url := range urls {
			_, err := fetcher.Fetch(url)
			if err != nil {
				b.Logf("Error fetching %s: %s, continuing", url.String(), err)
			}
		}
		err = dbm.Clear()
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkWarmupSqliteMemoryDB(b *testing.B) {
	b.Skip()
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	fetcher, err := makeFetcher(":memory:", ctx)
	if err != nil {
		b.Fatal(err)
	}
	urls, err := loadUrls()
	if err != nil {
		b.Fatal(err)
	}
	dbm, _ := fetcher.Storage.(store.Maintainable)
	for i := 0; i < b.N; i++ {
		for _, url := range urls {
			_, err := fetcher.Fetch(url)
			if err != nil {
				b.Logf("Error fetching %s: %s, continuing", url.String(), err)
			}
		}
		err = dbm.Clear()
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkWarmedSqliteMemoryDB(b *testing.B) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	fetcher, err := makeFetcher(":memory:", ctx)
	if err != nil {
		b.Fatal(err)
	}
	urls, err := loadUrls()
	if err != nil {
		b.Fatal(err)
	}
	for _, url := range urls {
		_, err := fetcher.Fetch(url)
		if err != nil {
			b.Logf("Error fetching %s: %s, continuing", url.String(), err)
		}
	}
	dbm, _ := fetcher.Storage.(store.Maintainable)
	for i := 0; i < b.N; i++ {
		for _, url := range urls {
			_, err := fetcher.Fetch(url)
			if err != nil {
				b.Logf("Error fetching %s: %s, continuing", url.String(), err)
			}
		}
		err = dbm.Clear()
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkWarmedSqliteFileDB(b *testing.B) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	fetcher, err := makeFetcher(":memory:", ctx)
	if err != nil {
		b.Fatal(err)
	}
	urls, err := loadUrls()
	if err != nil {
		b.Fatal(err)
	}
	for _, url := range urls {
		_, err := fetcher.Fetch(url)
		if err != nil {
			b.Logf("Error fetching %s: %s, continuing", url.String(), err)
		}
	}
	dbm, _ := fetcher.Storage.(store.Maintainable)
	for i := 0; i < b.N; i++ {
		for _, url := range urls {
			_, err := fetcher.Fetch(url)
			if err != nil {
				b.Logf("Error fetching %s: %s, continuing", url.String(), err)
			}
		}
		err = dbm.Clear()
		if err != nil {
			b.Fatal(err)
		}
	}
}
