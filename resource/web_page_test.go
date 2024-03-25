package resource

import (
	"bytes"
	"encoding/json"
	"errors"
	"io"
	nurl "net/url"
	"os"
	"testing"
	"time"
)

// Returns a WebPage will all fields filled out. The caller can override
// fields as needed.
func basicWebPage() WebPageNew {
	requestedUrl, _ := nurl.Parse("https://example.com/requested")
	canonicalUrl, _ := nurl.Parse("https://example.com/canonical")
	ttl := 30 * 24 * time.Hour
	fetchTime := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
	return WebPageNew{
		RequestedURL: requestedUrl,
		CanonicalURL: canonicalUrl,
		OriginalURL:  "https://example.com/original",
		TTL:          &ttl,
		FetchTime:    &fetchTime,
		Hostname:     "example.com",
		StatusCode:   200,
		Error:        errors.New("an error occurred"),
		Title:        "A title",
		Description:  "A description",
		Sitename:     "A sitename",
		Authors:      []string{"author1", "author2"},
		Date:         &fetchTime,
		Categories:   []string{"cat1", "cat2"},
		Tags:         []string{"tag1", "tag2"},
		Language:     "en",
		Image:        "https://example.com/image.jpg",
		PageType:     "article",
		License:      "CC-BY-SA",
		ID:           "1234",
		Fingerprint:  "fingerprint",
		ContentText:  "This is the content text",
	}
}

func TestMarshal(t *testing.T) {
	page := basicWebPage()
	var buf io.Writer
	var byteBuffer = new(bytes.Buffer)
	buf = io.MultiWriter(os.Stdout, byteBuffer)
	encoder := json.NewEncoder(buf)
	encoder.SetIndent("", "  ")
	encoder.Encode(page)
	// b, err := json.Marshal(page)
	// if err != nil {
	// 	t.Fatal(err)
	// }
	decoder := json.NewDecoder(byteBuffer)
	var rt WebPageNew
	err := decoder.Decode(&rt)
	if err != nil {
		t.Fatalf("Error decoding JSON: %s", err)
	}
	if err := isEqual(&page, &rt); err != nil {
		t.Errorf("Error comparing original and copy: %s", err)
	}
}

func isEqual(original, rt *WebPageNew) error {
	if original.RequestedURL == nil && rt.RequestedURL != nil {
		return errors.New("RequestedURL is present in original but nil in copy")
	}
	if original.CanonicalURL == nil && rt.CanonicalURL != nil {
		return errors.New("CanonicalURL is present in original but nil in copy")
	}
	if original.TTL != nil && rt.TTL == nil {
		return errors.New("TTL is present in original but nil in copy")
	}
	if original.FetchTime != nil && rt.FetchTime == nil {
		return errors.New("FetchTime is present in original but nil in copy")
	}
	if original.Date != nil && rt.Date == nil {
		return errors.New("Date is present in original but nil in copy")
	}
	return nil
}
