package trafilatura

import (
	"errors"
	nurl "net/url"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/efixler/scrape/fetch"
	"github.com/efixler/scrape/resource"
	"github.com/markusmobius/go-trafilatura"
)

func basicTrafilaturaResult() trafilatura.ExtractResult {
	return trafilatura.ExtractResult{
		ContentText: "T content text",
		Metadata: trafilatura.Metadata{
			URL:         "https://trafilatura.com/canonical",
			Title:       "T title",
			Author:      "author1;author2",
			Hostname:    "trafilatura.com",
			Description: "T description",
			Sitename:    "T sitename",
			Date:        time.Date(2023, 1, 1, 0, 0, 0, 0, time.UTC),
			Categories:  []string{"T cat1", "T cat2"},
			Tags:        []string{"T tag1", "T tag2"},
			Language:    "fr",
			Image:       "https://trafilatura.com/image.jpg",
			PageType:    "T article",
			License:     "T CC-BY-SA",
		},
	}
}

func TestMergeTrafilaturaResult(t *testing.T) {
	page := basicWebPage()
	tfc, _ := New(fetch.MustClient())
	tr := basicTrafilaturaResult()
	tfc.applyExtractResult(&tr, &page)
	if page.ContentText != tr.ContentText {
		t.Errorf("ContentText mismatch: %s != %s", page.ContentText, tr.ContentText)
	}
	if page.CanonicalURL.String() != tr.Metadata.URL {
		t.Errorf("CanonicalURL mismatch: %s != %s", page.CanonicalURL, tr.Metadata.URL)
	}
	if page.Title != tr.Metadata.Title {
		t.Errorf("Title mismatch: %s != %s", page.Title, tr.Metadata.Title)
	}
	if strings.Join(page.Authors, ";") != tr.Metadata.Author {
		t.Errorf("Authors mismatch: %v != %v", page.Authors, tr.Metadata.Author)
	}
	if page.Hostname != tr.Metadata.Hostname {
		t.Errorf("Hostname mismatch: %s != %s", page.Hostname, tr.Metadata.Hostname)
	}
	if page.Description != tr.Metadata.Description {
		t.Errorf("Description mismatch: %s != %s", page.Description, tr.Metadata.Description)
	}
	if page.Sitename != tr.Metadata.Sitename {
		t.Errorf("Sitename mismatch: %s != %s", page.Sitename, tr.Metadata.Sitename)
	}
	if page.Date.Compare(tr.Metadata.Date) != 0 {
		t.Errorf("Date mismatch: %s != %s", page.Date, tr.Metadata.Date)
	}
	if !slices.Equal(page.Categories, tr.Metadata.Categories) {
		t.Errorf("Categories mismatch: %v != %v", page.Categories, tr.Metadata.Categories)
	}
	if !slices.Equal(page.Tags, tr.Metadata.Tags) {
		t.Errorf("Tags mismatch: %v != %v", page.Tags, tr.Metadata.Tags)
	}
	if page.Language != tr.Metadata.Language {
		t.Errorf("Language mismatch: %s != %s", page.Language, tr.Metadata.Language)
	}
	if page.Image != tr.Metadata.Image {
		t.Errorf("Image mismatch: %s != %s", page.Image, tr.Metadata.Image)
	}
	if page.PageType != tr.Metadata.PageType {
		t.Errorf("PageType mismatch: %s != %s", page.PageType, tr.Metadata.PageType)
	}
	if page.FetchMethod != resource.DefaultClient {
		t.Errorf("FetchMethod should be set to default client, got: %s", page.FetchMethod)
	}
}

func TestEmptyAuthorNotSaved(t *testing.T) {
	page := basicWebPage()
	page.Authors = nil
	tfc, _ := New(fetch.MustClient())
	tr := basicTrafilaturaResult()
	tr.Metadata.Author = ""
	tfc.applyExtractResult(&tr, &page)
	if page.Authors == nil {
		t.Errorf("Authors was nil, expected empty array")
	}
	if len(page.Authors) != 0 {
		t.Errorf("Empty author should not be saved: %q", page.Authors)
	}
}

// Returns a WebPage will all fields filled out. The caller can override
// fields as needed.
func basicWebPage() resource.WebPage {
	requestedUrl, _ := nurl.Parse("https://example.com/requested")
	canonicalUrl, _ := nurl.Parse("https://example.com/canonical")
	fetchTime := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
	return resource.WebPage{
		RequestedURL: requestedUrl,
		CanonicalURL: canonicalUrl,
		OriginalURL:  "https://example.com/original",
		// TTL:          ttl, // skip ttl for now
		FetchTime:   &fetchTime,
		Hostname:    "example.com",
		StatusCode:  200,
		Error:       errors.New("an error occurred"),
		Title:       "A title",
		Description: "A description",
		Sitename:    "A sitename",
		Authors:     []string{"author1", "author2"},
		Date:        &fetchTime,
		Categories:  []string{"cat1", "cat2"},
		Tags:        []string{"tag1", "tag2"},
		Language:    "en",
		Image:       "https://example.com/image.jpg",
		PageType:    "article",
		License:     "CC-BY-SA",
		ID:          "1234",
		Fingerprint: "fingerprint",
		ContentText: "This is the content text",
	}
}
