package resource

import (
	"github.com/mmcdole/gofeed"
)

// Adds a RequestedURL field to the gofeed.Feed struct,
// along with the ItemLinks() function.
type Feed struct {
	RequestedURL string `json:"requested_url,omitempty"`
	gofeed.Feed
}

// Returns a slice of links for each item in the feed.
func (f Feed) ItemLinks() []string {
	rval := make([]string, len(f.Items))
	for i, item := range f.Items {
		rval[i] = item.Link
	}
	return rval
}
