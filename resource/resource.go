package resource

import (
	"encoding/json"
	"errors"
	nurl "net/url"
	"time"

	"github.com/markusmobius/go-trafilatura"
)

var (
	DefaultTTL = 30 * 24 * time.Hour
	nowf       = time.Now
)

type WebPage struct {
	trafilatura.Metadata
	OriginalURL  string         `json:"original_url,omitempty"` // The page that was requested by the caller
	RequestedURL *nurl.URL      `json:"-"`                      // The page that was actually fetched
	TTL          *time.Duration `json:"-"`
	FetchTime    *time.Time     `json:"fetch_time,omitempty"` // When the returned source was fetched
	StatusCode   int            `json:"status_code,omitempty"`
	Error        error          `json:"-"`
	ContentText  string         `json:"content_text,omitempty"`
	canonicalURL *nurl.URL
}

func (r WebPage) URL() *nurl.URL {
	if r.canonicalURL == nil {
		r.canonicalURL, _ = nurl.Parse(r.Metadata.URL)
	}
	return r.canonicalURL
}

func (r WebPage) MarshalJSON() ([]byte, error) {
	// Use this inline struct to control the output
	ar := &struct {
		URL                string     `json:"url,omitempty"`
		RequestedUrlString string     `json:"requested_url,omitempty"`
		OriginalURL        string     `json:"original_url,omitempty"`
		Hostname           string     `json:"hostname,omitempty"`
		FetchTime          *time.Time `json:"fetch_time,omitempty"`
		StatusCode         int        `json:"status_code,omitempty"`
		ErrorString        string     `json:"error,omitempty"`
		Title              string     `json:"title,omitempty"`
		Description        string     `json:"description,omitempty"`
		Author             string     `json:"author,omitempty"`
		Sitename           string     `json:"sitename,omitempty"`
		Date               *time.Time `json:"date,omitempty"`
		Categories         []string   `json:"categories,omitempty"`
		Tags               []string   `json:"tags,omitempty"`
		Language           string     `json:"language,omitempty"`
		Image              string     `json:"image,omitempty"`
		PageType           string     `json:"page_type,omitempty"`
		ID                 string     `json:"id,omitempty"`
		Fingerprint        string     `json:"fingerprint,omitempty"`
		License            string     `json:"license,omitempty"`
		ContentText        string     `json:"content_text,omitempty"`
	}{
		URL:         r.Metadata.URL,
		OriginalURL: r.OriginalURL,
		Hostname:    r.Hostname,
		FetchTime:   r.FetchTime,
		StatusCode:  r.StatusCode,
		Title:       r.Title,
		Description: r.Description,
		Author:      r.Author,
		Sitename:    r.Sitename,
		Categories:  r.Categories,
		Tags:        r.Tags,
		Language:    r.Language,
		Image:       r.Image,
		PageType:    r.PageType,
		ID:          r.ID,
		Fingerprint: r.Fingerprint,
		License:     r.License,
		ContentText: r.ContentText,
	}
	// We can control the output by clearing these fields
	// (in addition to ContentText.)
	if r.RequestedURL != nil {
		ar.RequestedUrlString = r.RequestedURL.String()
	}
	if r.Error != nil {
		ar.ErrorString = r.Error.Error()
	}
	if !r.Date.IsZero() {
		ar.Date = &r.Date
	}
	return json.Marshal(ar)
}

func (r *WebPage) UnmarshalJSON(b []byte) error {
	type alias WebPage
	// Unmarshal is case insensitive, so we only need to mask the fields
	// that we need to mutate, or when the name change is more than just case.
	ar := &struct {
		RequestedUrlString string `json:"requested_url,omitempty"`
		ErrorString        string `json:"error,omitempty"`
		PageType           string `json:"page_type,omitempty"`
		*alias
	}{
		alias: (*alias)(r),
	}
	if err := json.Unmarshal(b, ar); err != nil {
		return err
	}
	r.PageType = ar.PageType
	if ar.RequestedUrlString != "" {
		u, err := nurl.Parse(ar.RequestedUrlString)
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

func (r *WebPage) AssertTimes() {
	if r.FetchTime == nil || r.FetchTime.IsZero() {
		now := nowf().UTC().Truncate(time.Second)
		r.FetchTime = &now
	}
	if r.TTL == nil {
		ttl := DefaultTTL
		r.TTL = &ttl
	}
}
