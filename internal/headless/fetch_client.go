package headless

import (
	"net/http"

	"github.com/efixler/headless"
	"github.com/efixler/scrape/fetch"
)

type client struct {
	browser headless.TabFactory
}

func NewClient(browser headless.TabFactory) (fetch.Client, error) {
	c := &client{
		browser: browser,
	}
	return c, nil
}

func (c *client) Get(url string, headers http.Header) (*http.Response, error) {
	tab, err := c.browser.AcquireTab()
	if err != nil {
		return nil, err
	}
	return tab.Get(url, headers)
}
