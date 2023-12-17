package resource

import (
	"encoding/json"
	nurl "net/url"
	"time"

	"github.com/markusmobius/go-trafilatura"
)

type WebPage struct {
	trafilatura.Metadata
	ContentText  string     `json:",omitempty"`
	RequestedURL *nurl.URL  `json:"-"`
	FetchTime    *time.Time `json:",omitempty"`
	canonicalUrl *nurl.URL
}

func (r WebPage) URL() *nurl.URL {
	if r.canonicalUrl == nil {
		r.canonicalUrl, _ = nurl.Parse(r.Metadata.URL)
	}
	return r.canonicalUrl
}

func (r WebPage) MarshalJSON() ([]byte, error) {
	type alias WebPage
	ar := &struct {
		RequestedUrlString string `json:"RequestedURL,omitempty"`
		*alias
	}{
		alias:              (*alias)(&r),
		RequestedUrlString: "",
	}
	// We can control the output by clearing these fields
	// (in addition to ContentText.)
	if r.RequestedURL != nil {
		ar.RequestedUrlString = r.RequestedURL.String()
	}

	return json.Marshal(ar)
}
