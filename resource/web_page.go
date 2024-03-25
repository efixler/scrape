package resource

import (
	"encoding/json"
	"errors"
	nurl "net/url"
	"time"
)

// experimental: the original WebPage struct embeds trafilatura.Metadata,
// which was a nice way to start but had led to some confusing code.
// This struct is intended to replace that.

type WebPageNew struct { // The page that was requested by the caller
	RequestedURL *nurl.URL      `json:"-"` // The page that was actually fetched
	CanonicalURL *nurl.URL      `json:"-"`
	OriginalURL  string         `json:"original_url,omitempty"` // The canonical URL of the page
	TTL          *time.Duration `json:"-"`                      // Time to live for the resource
	FetchTime    *time.Time     `json:"fetch_time,omitempty"`   // When the returned source was fetched
	Hostname     string         `json:"hostname,omitempty"`     // Hostname of the page
	StatusCode   int            `json:"status_code,omitempty"`  // HTTP status code
	Error        error          `json:"error,omitempty"`
	Title        string         `json:"title,omitempty"`        // Title of the page
	Description  string         `json:"description,omitempty"`  // Description of the page
	Sitename     string         `json:"sitename,omitempty"`     // Name of the site
	Authors      []string       `json:"authors,omitempty"`      // Authors of the page
	Date         *time.Time     `json:"date,omitempty"`         // Date of the page
	Categories   []string       `json:"categories,omitempty"`   // Categories of the page
	Tags         []string       `json:"tags,omitempty"`         // Tags of the page
	Language     string         `json:"language,omitempty"`     // Language of the page
	Image        string         `json:"image,omitempty"`        // Image of the page
	PageType     string         `json:"page_type,omitempty"`    // Type of the page
	License      string         `json:"license,omitempty"`      // License of the page
	ID           string         `json:"id,omitempty"`           // ID of the page
	Fingerprint  string         `json:"fingerprint,omitempty"`  // Fingerprint of the page
	ContentText  string         `json:"content_text,omitempty"` // Error that occurred during fetching
}

func (r WebPageNew) MarshalJSON() ([]byte, error) {
	type alias WebPageNew
	ar := struct {
		URL                string `json:"url,omitempty"`
		RequestedURLString string `json:"requested_url,omitempty"`
		ErrorString        string `json:"error,omitempty"`
		*alias
	}{
		alias: (*alias)(&r),
	}
	if r.CanonicalURL != nil {
		ar.URL = r.CanonicalURL.String()
	}
	if r.RequestedURL != nil {
		ar.RequestedURLString = r.RequestedURL.String()
	}
	if r.Error != nil {
		ar.ErrorString = r.Error.Error()
	}
	return json.Marshal(ar)
}

func (r *WebPageNew) UnmarshalJSON(data []byte) error {
	type alias WebPageNew
	ar := struct {
		URL                string `json:"url,omitempty"`
		RequestedURLString string `json:"requested_url,omitempty"`
		ErrorString        string `json:"error,omitempty"`
		*alias
	}{
		alias: (*alias)(r),
	}
	if err := json.Unmarshal(data, &ar); err != nil {
		return err
	}
	if ar.URL != "" {
		u, err := nurl.Parse(ar.URL)
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
	return nil
}
