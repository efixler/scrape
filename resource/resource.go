package resource

import (
	"encoding/json"
	nurl "net/url"

	"github.com/markusmobius/go-trafilatura"
)

type WebPage struct {
	trafilatura.Metadata
	ContentText         string    `json:",omitempty"`
	RequestedURL        *nurl.URL `json:"-"`
	canonicalUrl        *nurl.URL
	noContentTextInJSON bool
}

func (r *WebPage) ContentTextInJSON(on bool) {
	r.noContentTextInJSON = !on
}

// type ResourceFetcher struct {
// 	store.UrlDataStore
// }

func (r WebPage) URL() *nurl.URL {
	if r.canonicalUrl == nil {
		r.canonicalUrl, _ = nurl.Parse(r.Metadata.URL)
	}
	return r.canonicalUrl
}

// func (r WebPage) UnmarshalJSON(data []byte) error {
// 	type alias WebPage
// 	ar := &struct {
// 		RequestedUrlString string `json:"RequestedURL"`
// 		*alias
// 	}{
// 		alias:              (*alias)(&r),
// 		RequestedUrlString: r.RequestedURL.String(),
// 	}
// 	//ar.RequestedUrl = r.RequestedURL.String()

// 	if err := json.Unmarshal(data, &ar); err != nil {
// 		return err
// 	}
// 	return json.Unmarshal(data, &ar)

// }

func (r WebPage) MarshalJSON() ([]byte, error) {
	type alias WebPage
	ar := &struct {
		RequestedUrlString string `json:"RequestedURL"`
		*alias
	}{
		alias:              (*alias)(&r),
		RequestedUrlString: r.RequestedURL.String(),
	}
	if r.noContentTextInJSON {
		ar.ContentText = ""
	}
	ar.RequestedUrlString = r.RequestedURL.String()

	return json.Marshal(ar)
}
