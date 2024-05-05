package fetch

import (
	"errors"
	"fmt"
	"log/slog"
	"net"
	"net/http"
	"path/filepath"
	"time"

	"github.com/efixler/scrape/resource"
)

const (
	DefaultTimeout = 30 * time.Second
)

type Client interface {
	Get(url string, headers http.Header) (*http.Response, error)
	Identifier() resource.FetchClient
}

type ClientOption func(*defaultClient) error

func MustClient(options ...ClientOption) Client {
	client, err := NewClient(options...)
	if err != nil {
		panic(err)
	}
	return client
}

func NewClient(options ...ClientOption) (Client, error) {
	client := &defaultClient{
		userAgent:  DefaultUserAgent,
		httpClient: &http.Client{Timeout: DefaultTimeout},
	}
	for _, opt := range options {
		if err := opt(client); err != nil {
			return nil, err
		}
	}
	return client, nil
}

type defaultClient struct {
	userAgent  string
	httpClient *http.Client
}

func (c defaultClient) Identifier() resource.FetchClient {
	return resource.DefaultClient
}

func (c defaultClient) Get(url string, headers http.Header) (*http.Response, error) {
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, err
	}
	if headers == nil {
		headers = make(http.Header)
	}
	req.Header = headers
	if req.Header.Get("User-Agent") == "" {
		req.Header.Set("User-Agent", c.userAgent)
	}
	slog.Debug("fetching", "url", url, "userAgent", c.userAgent)
	resp, err := c.httpClient.Do(req)
	if err != nil {
		if netErr, ok := err.(net.Error); ok && netErr.Timeout() {
			return nil, HttpError{
				StatusCode: http.StatusGatewayTimeout,
				Status:     http.StatusText(http.StatusGatewayTimeout),
				Message: fmt.Sprintf(
					"%s did not reply within %v seconds",
					url,
					c.httpClient.Timeout.Seconds(),
				),
			}
		}
		return resp, err
	}
	return resp, err
}

func WithTimeout(timeout time.Duration) ClientOption {
	return func(o *defaultClient) error {
		o.httpClient.Timeout = timeout
		return nil
	}
}

func WithUserAgent(ua string) ClientOption {
	return func(o *defaultClient) error {
		o.userAgent = ua
		return nil
	}
}

func WithFiles(path string) ClientOption {
	return func(o *defaultClient) error {
		if o.httpClient == nil {
			return errors.New("cannot use WithFiles with nil http.Client")
		}
		if o.httpClient.Transport == nil {
			o.httpClient.Transport = http.DefaultTransport
		}
		transport, ok := o.httpClient.Transport.(*http.Transport)
		if !ok {
			return errors.New("cannot use WithFiles with non-http.Transport")
		}
		abs, err := filepath.Abs(path)
		if err != nil {
			return err
		}
		transport.RegisterProtocol("file", http.NewFileTransport(http.Dir(abs)))
		return nil
	}
}

func WithHTTPClient(client *http.Client) ClientOption {
	return func(o *defaultClient) error {
		o.httpClient = client
		return nil
	}
}

func WithTransport(transport http.RoundTripper) ClientOption {
	return func(o *defaultClient) error {
		o.httpClient.Transport = transport
		return nil
	}
}
