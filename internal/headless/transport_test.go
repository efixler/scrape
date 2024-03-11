package headless

import (
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestAddressOption(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name      string
		address   string
		expectErr bool
	}{
		{
			name:      "test with address",
			address:   "http://localhost:8080",
			expectErr: false,
		},
		{
			name:      "test with empty address",
			address:   "",
			expectErr: true,
		},
	}
	for _, tt := range tests {
		o := &roundTripper{}
		if err := Address(tt.address)(o); (err != nil) != tt.expectErr {
			t.Fatalf("Address(%s) error = %v, wantErr %v", tt.name, err, tt.expectErr)
		}
		if o.headlessAddress != tt.address {
			t.Fatalf("Address(%s) headlessAddress = %v, want %v", tt.name, o.headlessAddress, tt.address)
		}
	}
}

func TestRoundTripperPostsToHeadless(t *testing.T) {
	srvr := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Fatalf("expected POST request, got %s", r.Method)
		}
		w.Header().Set("Content-Type", "text/plain")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("dummy headless response"))
	}))
	defer srvr.Close()
	rt, err := NewRoundTripper(Address(srvr.URL))
	if err != nil {
		t.Fatalf("NewRoundTripper error = %v", err)
	}
	req, err := http.NewRequest(http.MethodGet, "http://example.com", nil)
	if err != nil {
		t.Fatalf("http.NewRequest error = %v", err)
	}
	resp, err := rt.RoundTrip(req)
	if err != nil {
		t.Fatalf("RoundTrip error = %v", err)
	}
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, resp.StatusCode)
	}
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("io.ReadAll error = %v", err)
	}
	if string(body) != "dummy headless response" {
		t.Fatalf("expected body %q, got %q", "dummy headless response", string(body))
	}

}
