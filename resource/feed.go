package resource

import (
	"github.com/mmcdole/gofeed"
)

type Feed struct {
	RequestedURL string `json:"requested_url,omitempty"`
	gofeed.Feed
}

func (f Feed) ItemLinks() []string {
	rval := make([]string, len(f.Items))
	for i, item := range f.Items {
		rval[i] = item.Link
	}
	return rval
}
