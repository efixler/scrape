package trafilatura

import (
	"net/http"
	"path/filepath"
	"time"

	"github.com/efixler/scrape/fetch"
	"github.com/markusmobius/go-trafilatura"
)

var (
	DefaultTimeout      = 30 * time.Second
	trafilaturaFallback = &trafilatura.FallbackConfig{}
)

type Option func(*config) error

func defaultOptions() config {
	return config{
		FallbackConfig: &trafilatura.FallbackConfig{},
		HttpClient:     &http.Client{Timeout: DefaultTimeout},
		Timeout:        nil,
		Transport:      nil,
		UserAgent:      fetch.DefaultUserAgent,
	}
}

func WithClient(client *http.Client) Option {
	return func(o *config) error {
		o.HttpClient = client
		return nil
	}
}

// WithTimeout sets the timeout for the HTTP client.
func WithTimeout(timeout time.Duration) Option {
	return func(o *config) error {
		o.Timeout = &timeout
		return nil
	}
}

func WithUserAgent(ua string) Option {
	return func(o *config) error {
		o.UserAgent = ua
		return nil
	}
}

func WithFiles(path string) Option {
	return func(o *config) error {
		t := &http.Transport{}
		abs, err := filepath.Abs(path)
		if err != nil {
			return err
		}
		t.RegisterProtocol("file", http.NewFileTransport(http.Dir(abs)))
		o.Transport = t
		return nil
	}
}

func WithTransport(transport http.RoundTripper) Option {
	return func(o *config) error {
		o.Transport = transport
		return nil
	}
}
