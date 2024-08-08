// Provides a fetch.Client implementation that uses headless Chrome to fetch resources.
package headless

import (
	"context"
	"net/http"

	"github.com/efixler/headless"
	"github.com/efixler/headless/browser"
	"github.com/efixler/scrape/fetch"
	"github.com/efixler/scrape/resource"
)

type client struct {
	browser headless.TabFactory
}

func MustChromeClient(ctx context.Context, userAgent string, maxConcurrent int) fetch.Client {
	c, err := NewChromeClient(ctx, userAgent, maxConcurrent)
	if err != nil {
		panic(err)
	}
	return c
}

func NewChromeClient(ctx context.Context, userAgent string, maxConcurrent int) (fetch.Client, error) {
	browser, err := browser.NewChrome(
		ctx,
		browser.MaxTabs(maxConcurrent),
		browser.Headless(true),
		browser.AsFirefox(),
		browser.UserAgentIfNotEmpty(userAgent),
	)
	if err != nil {
		return nil, err
	}
	c := &client{
		browser: browser,
	}
	return c, nil
}

func (c client) Identifier() resource.ClientIdentifier {
	return resource.HeadlessChromium
}

func (c *client) Get(url string, headers http.Header) (*http.Response, error) {
	tab, err := c.browser.AcquireTab()
	if err != nil {
		return nil, err
	}
	return tab.Get(url, headers)
}
