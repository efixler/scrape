// Defines struct formats for web page and feed resources.
package resource

import (
	"encoding/json"
	"errors"
	nurl "net/url"
	"time"
)

type skippable string

const (
	CanonicalURL skippable = "canonical_url"
	ContentText  skippable = "content_text"
	OriginalURL  skippable = "original_url"
	FetchTime    skippable = "fetch_time"
	FetchMethod  skippable = "fetch_method"
	TTL          skippable = "ttl"
)

var (
	ErrNoTTL   = errors.New("TTL not set")
	DefaultTTL = 30 * 24 * time.Hour
)

func NewWebPage(url nurl.URL) *WebPage {
	fetchTime := time.Now().UTC().Truncate(time.Second)
	return &WebPage{
		RequestedURL: &url,
		FetchTime:    &fetchTime,
	}
}

// Represents a web page that was fetched, including metadata from the page itself,
// text content, and information about the fetch operation.
type WebPage struct { // The page that was requested by the caller
	RequestedURL *nurl.URL        `json:"-"` // The page that was actually fetched
	CanonicalURL *nurl.URL        `json:"-"`
	OriginalURL  string           `json:"original_url,omitempty"` // The canonical URL of the page
	TTL          time.Duration    `json:"-"`                      // Time to live for the resource
	FetchTime    *time.Time       `json:"fetch_time,omitempty"`   // When the returned source was fetched
	FetchMethod  ClientIdentifier `json:"fetch_method,omitempty"` // Method used to fetch the page
	Hostname     string           `json:"hostname,omitempty"`     // Hostname of the page
	StatusCode   int              `json:"status_code,omitempty"`  // HTTP status code
	Error        error            `json:"error,omitempty"`
	Title        string           `json:"title,omitempty"`        // Title of the page
	Description  string           `json:"description,omitempty"`  // Description of the page
	Sitename     string           `json:"sitename,omitempty"`     // Name of the site
	Authors      []string         `json:"authors,omitempty"`      // Authors of the page
	Date         *time.Time       `json:"date,omitempty"`         // Date of the page
	Categories   []string         `json:"categories,omitempty"`   // Categories of the page
	Tags         []string         `json:"tags,omitempty"`         // Tags of the page
	Language     string           `json:"language,omitempty"`     // Language of the page
	Image        string           `json:"image,omitempty"`        // Image of the page
	PageType     string           `json:"page_type,omitempty"`    // Type of the page
	License      string           `json:"license,omitempty"`      // License of the page
	ID           string           `json:"id,omitempty"`           // ID of the page
	Fingerprint  string           `json:"fingerprint,omitempty"`  // Fingerprint of the page
	ContentText  string           `json:"content_text,omitempty"` // Error that occurred during fetching
	skipMap      map[skippable]bool
}

func (r WebPage) ExpireTime() (time.Time, error) {
	if r.TTL == 0 {
		return time.Time{}, ErrNoTTL
	}
	t := r.FetchTime
	if t == nil {
		tt := time.Now()
		t = &tt
	}
	return t.Add(r.TTL), nil
}

func (r *WebPage) ClearSkipWhenMarshaling() {
	r.skipMap = nil
}

func (r *WebPage) SkipWhenMarshaling(skip ...skippable) {
	r.skipMap = make(map[skippable]bool)
	for _, s := range skip {
		r.skipMap[s] = true
	}
}

func (r WebPage) MarshalJSON() ([]byte, error) {
	type alias WebPage
	ar := struct {
		URLString          string `json:"url,omitempty"`
		RequestedURLString string `json:"requested_url,omitempty"`
		ErrorString        string `json:"error,omitempty"`
		alias
	}{
		alias: (alias)(r),
	}
	if r.CanonicalURL != nil {
		ar.URLString = r.CanonicalURL.String()
	}
	if r.RequestedURL != nil {
		ar.RequestedURLString = r.RequestedURL.String()
	}
	if r.Error != nil {
		ar.ErrorString = r.Error.Error()
	}
	if (r.Date != nil) && r.Date.IsZero() {
		ar.Date = nil
	}
	if (r.FetchTime != nil) && r.FetchTime.IsZero() {
		ar.FetchTime = nil
	}
	// Skip fields marked for skipping
	if r.skipMap != nil {
		for s := range r.skipMap {
			switch s {
			case CanonicalURL:
				ar.URLString = ""
			case ContentText:
				ar.ContentText = ""
			case OriginalURL:
				ar.OriginalURL = ""
			case FetchTime:
				ar.FetchTime = nil
			case FetchMethod:
				ar.FetchMethod = Unspecified
			case TTL:
				ar.TTL = 0
			}
		}
	}
	return json.Marshal(ar)
}

func (r *WebPage) UnmarshalJSON(data []byte) error {
	type alias WebPage
	ar := struct {
		URLString          string `json:"url,omitempty"`
		RequestedURLString string `json:"requested_url,omitempty"`
		ErrorString        string `json:"error,omitempty"`
		*alias
	}{
		alias: (*alias)(r),
	}
	if err := json.Unmarshal(data, &ar); err != nil {
		return err
	}
	if ar.URLString != "" {
		u, err := nurl.Parse(ar.URLString)
		if err != nil {
			return err
		}
		r.CanonicalURL = u
	}
	if ar.RequestedURLString != "" {
		u, err := nurl.Parse(ar.RequestedURLString)
		if err != nil {
			return err
		}
		r.RequestedURL = u
	}
	if ar.ErrorString != "" {
		r.Error = errors.New(ar.ErrorString)
	}
	if r.Date != nil && r.Date.IsZero() {
		r.Date = nil
	}
	if r.FetchTime != nil && r.FetchTime.IsZero() {
		r.FetchTime = nil
	}
	return nil
}
