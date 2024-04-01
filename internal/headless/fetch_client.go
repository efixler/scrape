package headless

import (
	"context"
	"net/http"

	"github.com/efixler/headless"
	"github.com/efixler/headless/browser"
	"github.com/efixler/scrape/fetch"
)

type client struct {
	browser headless.TabFactory
}

func MustChromeClient(ctx context.Context, maxConcurrent int) fetch.Client {
	c, err := NewChromeClient(ctx, maxConcurrent)
	if err != nil {
		panic(err)
	}
	return c
}

func NewChromeClient(ctx context.Context, maxConcurrent int) (fetch.Client, error) {
	browser, err := browser.NewChrome(
		ctx,
		browser.MaxTabs(maxConcurrent),
		browser.Headless(true),
		browser.AsFirefox(),
	)
	if err != nil {
		return nil, err
	}
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
