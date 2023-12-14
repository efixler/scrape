package resource

import (
	nurl "net/url"

	"github.com/markusmobius/go-trafilatura"
)

type WebPage struct {
	trafilatura.Metadata
	ContentText  string    `json:",omitempty"`
	ParsedUrl    *nurl.URL `json:"-"`
	canonicalUrl *nurl.URL `json:"-"`
}

func (r WebPage) URL() *nurl.URL {
	if r.canonicalUrl == nil {
		r.canonicalUrl, _ = nurl.Parse(r.Metadata.URL)
	}
	return r.canonicalUrl
}

// func (r *Resource) MarshalJSON() ([]byte, error) {
// 	return json.Marshal(r.Metadata)
// }
