package internal

import (
	"context"
	_ "embed"
	"encoding/json"
	"fmt"
	"html/template"
	"net/http"
	"net/http/httptest"
	nurl "net/url"
	"slices"
	"sync"
	"testing"
	"time"

	"github.com/efixler/scrape/database"
	"github.com/efixler/scrape/database/sqlite"
	"github.com/efixler/scrape/fetch"
	"github.com/efixler/scrape/fetch/trafilatura"
	"github.com/efixler/scrape/internal/storage"
	"github.com/efixler/scrape/resource"
)

//go:embed test_support/ars-2003724.html
var testPage []byte

//go:embed test_support/ars-2003724.json
var testJson []byte

func TestFetchStoresAndRetrieves(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		w.WriteHeader(http.StatusOK)
		w.Write(testPage)
	}))
	defer ts.Close()
	client := fetch.MustClient(fetch.WithHTTPClient(ts.Client()))
	tf, err := trafilatura.New(client)
	if err != nil {
		t.Fatal(err)
	}
	dbh := database.New(sqlite.MustNew(sqlite.InMemoryDB()))
	storage := storage.NewURLDataStore(dbh)

	fetcher, err := NewStorageBackedFetcher(tf, storage)
	if err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	err = fetcher.Open(ctx)
	if err != nil {
		t.Fatal(err)
	}
	baseUrl := ts.URL + "/arstechnica-1993804.html?keep_param=1"
	url := baseUrl + "&utm_source=feedburner"
	netURL, _ := nurl.Parse(url)
	fetched, err := fetcher.Fetch(netURL)
	if err != nil {
		t.Errorf("Expected no error for %s, got %s", url, err)
	}
	if fetched.OriginalURL != url {
		t.Errorf("Expected URL %s for fetched resource, got %s", url, fetched.OriginalURL)
	}
	if fetched.RequestedURL.String() != baseUrl {
		t.Errorf("Expected URL %s for fetched resource, got %s", baseUrl, fetched.RequestedURL.String())
	}
	if fetched.StatusCode != http.StatusOK {
		t.Errorf("Expected status code %d for fetched resource, got %d", http.StatusOK, fetched.StatusCode)
	}
	reference := &resource.WebPage{}
	err = json.Unmarshal(testJson, reference)
	if err != nil {
		t.Fatal(err)
	}
	time.Sleep(1 * time.Second) // make sure wallclock has advanced so we can check the fetch time
	stored, err := fetcher.Fetch(netURL)
	if err != nil {
		t.Errorf("Expected no error for %s, got %s", url, err)
	}

	for i, test := range []*resource.WebPage{fetched, stored} {
		var label string
		if i == 0 {
			label = "fetched"
		} else {
			label = "stored"
		}
		if test.StatusCode != reference.StatusCode {
			t.Errorf(
				"Expected status code %d for %s resource, got %d",
				reference.StatusCode,
				label,
				test.StatusCode)
		}
		if test.Title != reference.Title {
			t.Errorf("Expected title %s for %s resource, got %s", reference.Title, label, test.Title)
		}
		if test.Description != reference.Description {
			t.Errorf(
				"Expected description %s for %s resource, got %s",
				reference.Description,
				label,
				test.Description,
			)
		}
		if !slices.Equal(test.Authors, reference.Authors) {
			t.Errorf("Expected author %s for %s resource, got %s", reference.Authors, label, test.Authors)
		}
		if test.Sitename != reference.Sitename {
			t.Errorf("Expected sitename %s for %s resource, got %s", reference.Sitename, label, test.Sitename)
		}
		if test.Date.Compare(*reference.Date) != 0 {
			t.Errorf("Expected date %s for %s resource, got %s", reference.Date, label, test.Date)
		}
		if test.Language != reference.Language {
			t.Errorf("Expected language %s for %s resource, got %s", reference.Language, label, test.Language)
		}
		if test.Image != reference.Image {
			t.Errorf("Expected image %s for %s resource, got %s", reference.Image, label, test.Image)
		}
		if test.PageType != reference.PageType {
			t.Errorf("Expected page type %s for %s resource, got %s", reference.PageType, label, test.PageType)
		}
		if test.PageType != reference.PageType {
			t.Errorf("Expected page type %s for %s resource, got %s", reference.PageType, label, test.PageType)
		}
		if test.CanonicalURL.String() != reference.CanonicalURL.String() {
			t.Errorf("Expected URL %s for %s resource, got %s", reference.CanonicalURL.String(), label, test.CanonicalURL.String())
		}
		if test.Hostname != reference.Hostname {
			t.Errorf("Expected hostname %s for %s resource, got %s", reference.Hostname, label, test.Hostname)
		}
		if test.ContentText != reference.ContentText {
			t.Errorf("Expected content text %s for %s resource, got %s", reference.ContentText, label, test.ContentText)
		}
	}
	if !fetched.FetchTime.Equal(*stored.FetchTime) {
		t.Errorf("Expected fetch time %s for stored resource, got %s", fetched.FetchTime, stored.FetchTime)
	}
}

func TestFetchUnstored(t *testing.T) {
	tmpl, _ := htmlTemplate()
	var ts *httptest.Server
	ts = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		w.WriteHeader(http.StatusOK)
		data := struct {
			URL    string
			Number int64
		}{
			URL:    fmt.Sprintf("%s%s", ts.URL, r.URL.String()),
			Number: time.Now().UnixNano(),
		}
		err := tmpl.Execute(w, data)
		if err != nil {
			t.Fatal(err)
		}
	}))
	defer ts.Close()

	client := fetch.MustClient(fetch.WithHTTPClient(ts.Client()))
	tf, err := trafilatura.New(client)
	if err != nil {
		t.Fatal(err)
	}
	dbh := database.New(sqlite.MustNew(sqlite.InMemoryDB()))
	storage := storage.NewURLDataStore(dbh)
	fetcher, err := NewStorageBackedFetcher(tf, storage)
	if err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	err = fetcher.Open(ctx)
	if err != nil {
		t.Fatal(err)
	}
	numpages := 10
	urls := make([]string, numpages)
	msgs := make([]fetchMsg, numpages)
	pageChan := make(chan *resource.WebPage, numpages)
	fetchChan := make(chan fetchMsg, numpages)
	for i := 1; i <= numpages; i++ {
		url := fmt.Sprintf("%s/%d", ts.URL, i)
		urls[i-1] = url
		netURL, _ := nurl.Parse(url)
		msg := fetchMsg{cleanedURL: netURL, originalURL: url}
		msgs[i-1] = msg
		fetchChan <- msg
	}
	close(fetchChan)
	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		fetcher.fetchUnstored(fetchChan, pageChan)
	}()
	wg.Wait()
	close(pageChan)
	var j = 0
	for page := range pageChan {
		if page.Error != nil {
			t.Errorf("Expected no error for %s, got %s", page.RequestedURL, page.Error)
		}
		if page.OriginalURL != msgs[j].originalURL {
			t.Errorf("Expected URL %s for fetched resource, got %s", msgs[j].originalURL, page.OriginalURL)
		}
		if page.RequestedURL.String() != msgs[j].cleanedURL.String() {
			t.Errorf("Expected URL %s for fetched resource, got %s", msgs[j].cleanedURL.String(), page.RequestedURL.String())
		}
		if page.StatusCode != http.StatusOK {
			t.Errorf("Expected status code %d for fetched resource, got %d", http.StatusOK, page.StatusCode)
		}
		j++
	}
	if j != numpages {
		t.Errorf("Expected %d pages to be fetched, got %d", numpages, j)
	}
	// sleep a little here to make sure the async stores finish.
	// these are only at risk in fast/unexpected exits, but do need to fix
	time.Sleep(10 * time.Millisecond)
	pageChan = make(chan *resource.WebPage, numpages)
	fetchChan = make(chan fetchMsg, numpages)
	wg.Add(1)
	// we can use loadBatch here to verify that fetchUnstored worked
	go func() {
		defer wg.Done()
		fetcher.loadBatch(urls, pageChan, fetchChan)
	}()
	wg.Wait()
	close(pageChan)
	var notFoundCount = 0
	for range fetchChan {
		notFoundCount++
	}
	if notFoundCount != 0 {
		t.Errorf("Expected no pages to be not found, got %d", notFoundCount)
	}
	var k = 0
	for page := range pageChan {
		if page.Error != nil {
			t.Errorf("Expected no error for %s, got %s", page.RequestedURL, page.Error)
		}
		if page.OriginalURL != msgs[k].originalURL {
			t.Errorf("Expected URL %s for fetched resource, got %s", msgs[k].originalURL, page.OriginalURL)
		}
		if page.RequestedURL.String() != msgs[k].cleanedURL.String() {
			t.Errorf("Expected URL %s for fetched resource, got %s", msgs[k].cleanedURL.String(), page.RequestedURL.String())
		}
		if page.StatusCode != http.StatusOK {
			t.Errorf("Expected status code %d for fetched resource, got %d", http.StatusOK, page.StatusCode)
		}
		k++
	}
	if k != numpages {
		t.Errorf("Expected %d pages to be fetched, got %d", numpages, k)
	}
}

func htmlTemplate() (*template.Template, error) {
	templateHTML := `
        <html>
            <head>
                <title>Web Page Number {{.Number}}</title>
                <meta property="og:url" content="{{.URL}}">
            </head>
            <body>
                <h1>Hello, World!</h1>
                <p>This is a randomly generated HTML document.</p>
                <p>Random number: {{.Number}}</p>
            </body>
        </html>
    `
	tmpl, err := template.New("fake_html").Parse(templateHTML)
	if err != nil {
		fmt.Println("Error parsing template:", err)
		return nil, err
	}
	return tmpl, nil
}
