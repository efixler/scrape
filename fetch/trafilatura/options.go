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

type option func(*Config) error

func defaultOptions() Config {
	return Config{
		FallbackConfig: &trafilatura.FallbackConfig{},
		HttpClient:     &http.Client{Timeout: DefaultTimeout},
		Timeout:        nil,
		Transport:      nil,
		UserAgent:      fetch.DefaultUserAgent,
	}
}

func WithClient(client *http.Client) option {
	return func(o *Config) error {
		o.HttpClient = client
		return nil
	}
}

// WithTimeout sets the timeout for the HTTP client.
func WithTimeout(timeout time.Duration) option {
	return func(o *Config) error {
		o.Timeout = &timeout
		return nil
	}
}

func WithUserAgent(ua string) option {
	return func(o *Config) error {
		o.UserAgent = ua
		return nil
	}
}

func WithFiles(path string) option {
	return func(o *Config) error {
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

func WithTransport(transport http.RoundTripper) option {
	return func(o *Config) error {
		o.Transport = transport
		return nil
	}
}
