package trafilatura

import (
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
		o := defaultOptions()
		if err := WithClient(tt.client)(&o); (err != nil) != tt.expectErr {
			t.Fatalf("WithClient(%s) error = %v, wantErr %v", tt.name, err, tt.expectErr)
		}
		tf, err := Factory(WithClient(tt.client))()
		if err != nil {
			t.Fatalf("Factory(%s) error = %v", tt.name, err)
		}
		if tf.(*TrafilaturaFetcher).httpClient != tt.client {
			t.Fatalf("Factory(%s) httpClient = %v, want %v", tt.name, tf.(*TrafilaturaFetcher).httpClient, tt.client)
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
		o := defaultOptions()
		if err := WithTimeout(tt.timeout)(&o); (err != nil) != tt.expectErr {
			t.Fatalf("WithTimeout(%s) error = %v, wantErr %v", tt.name, err, tt.expectErr)
		}
		tf, err := Factory(WithTimeout(tt.timeout))()
		if err != nil {
			t.Fatalf("Factory(%s) error = %v", tt.name, err)
		}
		if tf.(*TrafilaturaFetcher).httpClient.Timeout != tt.timeout {
			t.Fatalf("Factory(%s) httpClient.Timeout = %v, want %v", tt.name, tf.(*TrafilaturaFetcher).httpClient.Timeout, tt.timeout)
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
		o := defaultOptions()
		if err := WithUserAgent(tt.userAgent)(&o); (err != nil) != tt.expectErr {
			t.Fatalf("WithUserAgent(%s) error = %v, wantErr %v", tt.name, err, tt.expectErr)
		}
		tf, err := Factory(WithUserAgent(tt.userAgent))()
		if err != nil {
			t.Fatalf("Factory(%s) error = %v", tt.name, err)
		}
		if tf.(*TrafilaturaFetcher).userAgent != tt.userAgent {
			t.Fatalf("Factory(%s) userAgent = %v, want %v", tt.name, tf.(*TrafilaturaFetcher).userAgent, tt.userAgent)
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
	// TODO: Test that a transport works as expected
	for _, tt := range tests {
		o := defaultOptions()
		if err := WithTransport(tt.transport)(&o); (err != nil) != tt.expectErr {
			t.Fatalf("WithTransport(%s) error = %v, wantErr %v", tt.name, err, tt.expectErr)
		}
		tf, err := Factory(WithTransport(tt.transport))()
		if err != nil {
			t.Fatalf("Factory(%s) error = %v", tt.name, err)
		}
		if tf.(*TrafilaturaFetcher).httpClient.Transport != tt.transport {
			t.Fatalf("Factory(%s) httpClient.Transport = %v, want %v", tt.name, tf.(*TrafilaturaFetcher).httpClient.Transport, tt.transport)
		}
	}
}
