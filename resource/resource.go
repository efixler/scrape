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
	ar := &struct {
		RequestedUrlString string `json:"RequestedURL,omitempty"`
		ErrorString        string `json:"Error,omitempty"`
		*alias
	}{
		alias:              (*alias)(&r),
		RequestedUrlString: "",
		ErrorString:        "",
	}
	// We can control the output by clearing these fields
	// (in addition to ContentText.)
	if r.RequestedURL != nil {
		ar.RequestedUrlString = r.RequestedURL.String()
	}
	// else if r.OriginalURL != "" {
	// 	ar.RequestedUrlString = r.OriginalURL
	// }
	if r.Error != nil {
		ar.ErrorString = r.Error.Error()
	}
	return json.Marshal(ar)
}
