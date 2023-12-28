package scrape

import (
	"context"
	"encoding/csv"
	"io"
	"math/rand"
	nurl "net/url"
	"os"
	"testing"

	"github.com/efixler/scrape/fetch/trafilatura"
	"github.com/efixler/scrape/store"
	"github.com/efixler/scrape/store/sqlite"
)

const (
	CsvFile = "./internal/testdata/global_urls.csv"
	MaxURLs = 100
)

var urls = make([]*nurl.URL, 0, MaxURLs)

func init() {
	csvFile, err := os.Open(CsvFile)
	if err != nil {
		panic(err)
	}
	defer csvFile.Close()
	reader := csv.NewReader(csvFile)
	reader.FieldsPerRecord = -1 // allow variable number of fields, we only care about the first
	reader.TrimLeadingSpace = true
	reader.ReuseRecord = true
	allURLs := make([]*nurl.URL, 0, 2000)
	for {
		record, err := reader.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			panic(err)
		}
		url, err := nurl.Parse(record[1])
		if err != nil {
			continue
		}
		allURLs = append(allURLs, url)
	}
	segment := rand.Intn(len(allURLs) / MaxURLs)
	urls = allURLs[segment*MaxURLs : (segment+1)*MaxURLs]
}

func makeFetcher(dbPath string, ctx context.Context) (*StorageBackedFetcher, error) {
	fetcher, err := NewStorageBackedFetcher(
		trafilatura.Factory(),
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

func warmup(fetcher *StorageBackedFetcher, urls []*nurl.URL, b *testing.B) {
	for _, url := range urls {
		_, err := fetcher.Fetch(url)
		if err != nil {
			b.Logf("Error fetching %s: %s, continuing", url.String(), err)
		}
	}
}

func BenchmarkSqliteFileDBWarmedUp(b *testing.B) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	fetcher, err := makeFetcher(sqlite.DefaultDatabase, ctx)
	dbm, _ := fetcher.Storage.(store.Maintainable)
	defer dbm.Clear()
	if err != nil {
		panic(err)
	}
	warmup(fetcher, urls, b)
	for i := 0; i < b.N; i++ {
		for _, url := range urls {
			_, err := fetcher.Fetch(url)
			if err != nil {
				b.Logf("Error fetching %s: %s, continuing", urls[i], err)
			}
		}
	}
}

func BenchmarkSqliteMemoryDBWarmedUp(b *testing.B) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	fetcher, err := makeFetcher(sqlite.InMemoryDBName, ctx)
	if err != nil {
		panic(err)
	}
	warmup(fetcher, urls, b)
	for i := 0; i < b.N; i++ {
		for _, url := range urls {
			_, err := fetcher.Fetch(url)
			if err != nil {
				b.Logf("Error fetching %s: %s, continuing", url, err)
			}
		}
	}
}
