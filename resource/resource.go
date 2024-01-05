package resource

import (
	"encoding/json"
	nurl "net/url"
	"time"

	"github.com/markusmobius/go-trafilatura"
)

type WebPage struct {
	trafilatura.Metadata
	// The page that was requested by the caller
	OriginalURL string `json:",omitempty"`
	// When the returned source was fetched
	FetchTime   *time.Time `json:",omitempty"`
	StatusCode  int        `json:",omitempty"`
	Error       error      `json:"-"`
	ContentText string     `json:",omitempty"`
	// The url that was requested by whatever locally was fetching: allows
	// for something downstream from OriginalURL to filter query params
	RequestedURL *nurl.URL `json:"-"`
	canonicalURL *nurl.URL
}

func (r WebPage) URL() *nurl.URL {
	if r.canonicalURL == nil {
		r.canonicalURL, _ = nurl.Parse(r.Metadata.URL)
	}
	return r.canonicalURL
}

func (r WebPage) MarshalJSON() ([]byte, error) {
	type alias WebPage
	// This alias is mainly here to precisely control the JSON output.
	ar := &struct {
		OriginalURL        string     `json:"OriginalURL,omitempty"`
		RequestedUrlString string     `json:"RequestedURL,omitempty"`
		FetchTime          *time.Time `json:"FetchTime,omitempty"`
		StatusCode         int        `json:"StatusCode,omitempty"`
		ErrorString        string     `json:"Error,omitempty"`
		Title              string     `json:"Title,omitempty"`
		Description        string     `json:"Description,omitempty"`
		Author             string     `json:"Author,omitempty"`
		Sitename           string     `json:"Sitename,omitempty"`
		Date               *time.Time `json:"Date,omitempty"`
		Categories         []string   `json:"Categories,omitempty"`
		Tags               []string   `json:"Tags,omitempty"`
		Language           string     `json:"Language,omitempty"`
		Image              string     `json:"Image,omitempty"`
		PageType           string     `json:"PageType,omitempty"`
		ID                 string     `json:"ID,omitempty"`
		Fingerprint        string     `json:"Fingerprint,omitempty"`
		License            string     `json:"License,omitempty"`
		*alias
	}{
		alias:       (*alias)(&r),
		OriginalURL: r.OriginalURL,
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
