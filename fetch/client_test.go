package fetch

import (
	"errors"
	"net/http"
	"testing"
	"time"
)

func TestWithClient(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name      string
		client    *http.Client
		expectErr bool
	}{
		{
			name:      "test with client",
			client:    &http.Client{},
			expectErr: false,
		},
	}
	for _, tt := range tests {
		o, err := NewClient(WithHTTPClient(tt.client))
		if (err != nil) != tt.expectErr {
			t.Fatalf("WithHTTPClient(%s) error = %v, wantErr %v", tt.name, err, tt.expectErr)
		}
		if o.(*defaultClient).httpClient != tt.client {
			t.Fatalf("Factory(%s) httpClient = %v, want %v", tt.name, o.(*defaultClient).httpClient, tt.client)
		}
	}
}

func TestWithTimeout(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name      string
		timeout   time.Duration
		expectErr bool
	}{
		{
			name:      "test with timeout",
			timeout:   1 * time.Second,
			expectErr: false,
		},
	}
	for _, tt := range tests {
		o, err := NewClient(WithTimeout(tt.timeout))
		if (err != nil) != tt.expectErr {
			t.Fatalf("WithTimeout(%s) error = %v, wantErr %v", tt.name, err, tt.expectErr)
		}
		if o.(*defaultClient).httpClient.Timeout != tt.timeout {
			t.Fatalf("defaultClient(%s) httpClient.Timeout = %v, want %v", tt.name, o.(*defaultClient).httpClient.Timeout, tt.timeout)
		}
	}
}

func TestWithUserAgent(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name      string
		userAgent string
		expectErr bool
	}{
		{
			name:      "test with user agent",
			userAgent: "test user agent",
			expectErr: false,
		},
	}
	// TODO: test that request actually has user agent
	for _, tt := range tests {

		o, err := NewClient(WithUserAgent(tt.userAgent))
		if (err != nil) != tt.expectErr {
			t.Fatalf("WithUserAgent(%s) error = %v, wantErr %v", tt.name, err, tt.expectErr)
		}
		if o.(*defaultClient).userAgent != tt.userAgent {
			t.Fatalf("Factory(%s) userAgent = %v, want %v", tt.name, o.(*defaultClient).userAgent, tt.userAgent)
		}
	}
}

func TestWithTransport(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name      string
		transport http.RoundTripper
		expectErr bool
	}{
		{
			name:      "test with transport",
			transport: &http.Transport{},
			expectErr: false,
		},
	}
	for _, tt := range tests {
		o, err := NewClient(WithTransport(tt.transport))
		if (err != nil) != tt.expectErr {
			t.Fatalf("WithTransport(%s) error = %v, wantErr %v", tt.name, err, tt.expectErr)
		}
		if o.(*defaultClient).httpClient.Transport != tt.transport {
			t.Fatalf("Factory(%s) httpClient.Transport = %v, want %v", tt.name, o.(*defaultClient).httpClient.Transport, tt.transport)
		}
	}
}

type roundTripperFunc func(*http.Request) (*http.Response, error)

func (f roundTripperFunc) RoundTrip(req *http.Request) (*http.Response, error) {
	return f(req)
}

func TestWithFilesInvalidTransport(t *testing.T) {
	t.Parallel()

	rtf := func(*http.Request) (*http.Response, error) {
		return nil, errors.New("fake")
	}
	fakeRT := roundTripperFunc(rtf)
	_, err := NewClient(WithTransport(fakeRT), WithFiles("test"))
	if err == nil {
		t.Fatalf("WithFiles() error = %v, wantErr %v", err, true)
	}
}
