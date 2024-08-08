package resource

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	nurl "net/url"
	"os"
	"slices"
	"testing"
	"time"
)

// Returns a WebPage will all fields filled out. The caller can override
// fields as needed.
func basicWebPage() WebPage {
	requestedUrl, _ := nurl.Parse("https://example.com/requested")
	canonicalUrl, _ := nurl.Parse("https://example.com/canonical")
	fetchTime := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
	return WebPage{
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
		FetchMethod: DefaultClient,
	}
}

func TestMarshalBaseCase(t *testing.T) {
	page := basicWebPage()
	var buf io.Writer
	var byteBuffer = new(bytes.Buffer)
	buf = io.MultiWriter(os.Stdout, byteBuffer)
	encoder := json.NewEncoder(buf)
	encoder.SetIndent("", "  ")
	encoder.Encode(page)
	decoder := json.NewDecoder(byteBuffer)
	var rt WebPage
	err := decoder.Decode(&rt)
	if err != nil {
		t.Fatalf("Error decoding JSON: %s", err)
	}
	if err := isEqual(&page, &rt); err != nil {
		t.Errorf("Error comparing original and copy: %s", err)
	}
}

func TestMarshalDateZero(t *testing.T) {
	page := basicWebPage()
	zeroTime := time.Time{}
	page.FetchTime = &zeroTime
	page.Date = &zeroTime
	var buf io.Writer
	var byteBuffer = new(bytes.Buffer)
	buf = io.MultiWriter(os.Stdout, byteBuffer)
	encoder := json.NewEncoder(buf)
	encoder.SetIndent("", "  ")
	encoder.Encode(page)
	decoder := json.NewDecoder(byteBuffer)
	var rt WebPage
	err := decoder.Decode(&rt)
	if err != nil {
		t.Fatalf("Error decoding JSON: %s", err)
	}
	if rt.FetchTime != nil {
		t.Errorf("Round trip FetchTime expected nil from zero date, got %v", rt.FetchTime)
	}
	if rt.Date != nil {
		t.Errorf("Round trip Date expected nil from zero date, got %v", rt.Date)
	}
}

func TestSkipWhenMarshalling(t *testing.T) {
	page := basicWebPage()
	page.SkipWhenMarshaling(CanonicalURL, ContentText, FetchTime, FetchMethod, OriginalURL)
	var byteBuffer = new(bytes.Buffer)
	encoder := json.NewEncoder(byteBuffer)
	encoder.SetIndent("", "  ")
	encoder.Encode(page)
	decoder := json.NewDecoder(byteBuffer)
	var rt WebPage
	err := decoder.Decode(&rt)
	if err != nil {
		t.Fatalf("Error decoding JSON: %s", err)
	}
	if rt.CanonicalURL != nil {
		t.Errorf("Round trip CanonicalURL expected nil, got %v", rt.CanonicalURL)
	}
	if rt.ContentText != "" {
		t.Errorf("Round trip ContentText expected empty string, got %s", rt.ContentText)
	}
	if rt.OriginalURL != "" {
		t.Errorf("Round trip OriginalURL expected empty string, got %s", rt.OriginalURL)
	}
	if rt.FetchTime != nil {
		t.Errorf("Round trip FetchTime expected nil, got %v", rt.FetchTime)
	}
	if rt.FetchMethod != Unspecified {
		t.Errorf("Round trip FetchMethod expected Unspecified, got %v", rt.FetchMethod)
	}
	page.SkipWhenMarshaling()
	byteBuffer.Reset()
	encoder.Encode(page)
	decoder = json.NewDecoder(byteBuffer)
	err = decoder.Decode(&rt)
	if err != nil {
		t.Fatalf("Error decoding JSON: %s", err)
	}
	if rt.CanonicalURL.String() != page.CanonicalURL.String() {
		t.Errorf("Round trip CanonicalURL expected %v, got %v", page.CanonicalURL, rt.CanonicalURL)
	}
	if rt.ContentText != page.ContentText {
		t.Errorf("Round trip ContentText expected %s, got %s", page.ContentText, rt.ContentText)
	}
	if rt.FetchTime.Compare(*page.FetchTime) != 0 {
		t.Errorf("Round trip FetchTime expected %s, got %s", page.FetchTime, rt.FetchTime)
	}
	if rt.FetchMethod != page.FetchMethod {
		t.Errorf("Round trip FetchMethod expected %v, got %v", page.FetchMethod, rt.FetchMethod)
	}
	if rt.OriginalURL != page.OriginalURL {
		t.Errorf("Round trip OriginalURL expected %s, got %s", page.OriginalURL, rt.OriginalURL)
	}
}

func isEqual(original, rt *WebPage) error {
	if (original.RequestedURL == nil) != (rt.RequestedURL == nil) {
		return fmt.Errorf("RequestedURL nil mistmatch: %v != %v", original.RequestedURL, rt.RequestedURL)
	}
	if (original.CanonicalURL == nil) != (rt.CanonicalURL == nil) {
		return fmt.Errorf("CanonicalURL nil mistmatch: %v != %v", original.CanonicalURL, rt.CanonicalURL)
	}
	if (original.FetchTime == nil) != (rt.FetchTime == nil) {
		return fmt.Errorf("FetchTime nil mistmatch: %v != %v", original.FetchTime, rt.FetchTime)
	}
	if (original.Error == nil) != (rt.Error == nil) {
		return fmt.Errorf("Error nil mistmatch: %v != %v", original.Error, rt.Error)
	}
	if (original.Date == nil) != (rt.Date == nil) {
		return fmt.Errorf("Date nil mistmatch: %v != %v", original.Date, rt.Date)
	}
	if original.OriginalURL != rt.OriginalURL {
		return fmt.Errorf("OriginalURL mismatch: %s != %s", original.OriginalURL, rt.OriginalURL)
	}
	if original.CanonicalURL.String() != rt.CanonicalURL.String() {
		return fmt.Errorf("CanonicalURL mismatch: %s != %s", original.CanonicalURL, rt.CanonicalURL)
	}
	if original.TTL != rt.TTL {
		return fmt.Errorf("TTL mismatch: %s != %s", original.TTL, rt.TTL)
	}
	if original.FetchTime.Compare(*rt.FetchTime) != 0 {
		return fmt.Errorf("FetchTime mismatch: %s != %s", original.FetchTime, rt.FetchTime)
	}
	if original.Hostname != rt.Hostname {
		return fmt.Errorf("Hostname mismatch: %s != %s", original.Hostname, rt.Hostname)
	}
	if original.StatusCode != rt.StatusCode {
		return fmt.Errorf("StatusCode mismatch: %d != %d", original.StatusCode, rt.StatusCode)
	}
	if original.Error.Error() != rt.Error.Error() {
		return fmt.Errorf("Error mismatch: %s != %s", original.Error, rt.Error)
	}
	if original.Title != rt.Title {
		return fmt.Errorf("Title mismatch: %s != %s", original.Title, rt.Title)
	}
	if original.Description != rt.Description {
		return fmt.Errorf("Description mismatch: %s != %s", original.Description, rt.Description)
	}
	if !slices.Equal(original.Authors, rt.Authors) {
		return fmt.Errorf("Authors mismatch: %v != %v", original.Authors, rt.Authors)
	}
	if original.Date.Compare(*rt.Date) != 0 {
		return fmt.Errorf("Date mismatch: %s != %s", original.Date, rt.Date)
	}
	if !slices.Equal(original.Categories, rt.Categories) {
		return fmt.Errorf("Categories mismatch: %v != %v", original.Categories, rt.Categories)
	}
	if !slices.Equal(original.Tags, rt.Tags) {
		return fmt.Errorf("Tags mismatch: %v != %v", original.Tags, rt.Tags)
	}
	if original.Language != rt.Language {
		return fmt.Errorf("Language mismatch: %s != %s", original.Language, rt.Language)
	}
	if original.Image != rt.Image {
		return fmt.Errorf("Image mismatch: %s != %s", original.Image, rt.Image)
	}
	if original.PageType != rt.PageType {
		return fmt.Errorf("PageType mismatch: %s != %s", original.PageType, rt.PageType)
	}
	if original.License != rt.License {
		return fmt.Errorf("License mismatch: %s != %s", original.License, rt.License)
	}
	if original.ID != rt.ID {
		return fmt.Errorf("ID mismatch: %s != %s", original.ID, rt.ID)
	}
	if original.Fingerprint != rt.Fingerprint {
		return fmt.Errorf("Fingerprint mismatch: %s != %s", original.Fingerprint, rt.Fingerprint)
	}
	if original.ContentText != rt.ContentText {
		return fmt.Errorf("ContentText mismatch: %s != %s", original.ContentText, rt.ContentText)
	}
	if original.FetchMethod != rt.FetchMethod {
		return fmt.Errorf("FetchMethod mismatch: %s != %s", original.FetchMethod, rt.FetchMethod)
	}
	return nil
}

func TestExpireTime(t *testing.T) {
	page := basicWebPage()
	page.TTL = 24 * time.Hour
	expireTime, err := page.ExpireTime()
	if err != nil {
		t.Fatalf("Error getting expire time: %s", err)
	}
	expectedTime := page.FetchTime.Add(page.TTL)
	if expireTime != expectedTime {
		t.Errorf("ExpireTime mismatch: %s != %s", expireTime, expectedTime)
	}
	page.TTL = 0
	_, err = page.ExpireTime()
	if err != ErrNoTTL {
		t.Errorf("Expected ErrNoTTL, got %s", err)
	}
}

func TestFetchMethod(t *testing.T) {
	tests := []struct {
		name string
		f    ClientIdentifier
		want string
	}{
		{
			name: "Client",
			f:    DefaultClient,
			want: "direct",
		},
		{
			name: "Headless",
			f:    HeadlessChromium,
			want: "chromium-headless",
		},
		{
			name: "Unknown",
			f:    0,
			want: "unspecified",
		},
	}
	for _, tt := range tests {
		page := basicWebPage()
		page.FetchMethod = tt.f
		var byteBuffer = new(bytes.Buffer)
		encoder := json.NewEncoder(byteBuffer)
		encoder.SetIndent("", "  ")
		err := encoder.Encode(page)
		if err != nil {
			t.Fatalf("error encoding JSON: %v", err)
		}
		decoder := json.NewDecoder(byteBuffer)
		var rt WebPage
		err = decoder.Decode(&rt)
		if err != nil {
			t.Fatalf("Error decoding JSON: %s", err)
		}
		if got := page.FetchMethod.String(); got != tt.want {
			t.Errorf("[%s] page.FetchMethod.String() = %v, want %v", tt.name, got, tt.want)
		}
	}
}
