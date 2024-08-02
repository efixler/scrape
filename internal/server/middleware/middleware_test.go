package middleware

import (
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestMaxBytes(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name      string
		n         int64
		body      string
		expectErr bool
	}{
		{
			name:      "too big",
			n:         1,
			body:      "test",
			expectErr: true,
		},
		{
			name:      "just right",
			n:         4,
			body:      "test",
			expectErr: false,
		},
	}
	for _, tt := range tests {
		req := httptest.NewRequest("POST", "http://example.com", strings.NewReader(tt.body))
		w := httptest.NewRecorder()
		m := MaxBytes(tt.n)
		m(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			_, err := io.ReadAll(r.Body)
			if (err != nil) != tt.expectErr {
				t.Fatalf("[%s] MaxBytesReader, expected error %v, got %v", tt.name, tt.expectErr, err)
			}
		}))(w, req)
	}
}

func TestDecodeJSONBody(t *testing.T) {
	t.Parallel()
	type payloadKey struct{}

	type payload struct {
		Urls []string `json:"urls"`
		Msg  string   `json:"msg,omitempty"`
	}

	tests := []struct {
		name         string
		body         string
		expectStatus int
	}{
		{
			name:         "valid",
			body:         `{"urls":["http://example.com"]}`,
			expectStatus: 200,
		},
		{
			name:         "invalid unknown field",
			body:         `{"url":["http://example.com"]}`,
			expectStatus: 400,
		},
		{
			name:         "invalid bad json",
			body:         `{"urls":[["http://example.com"]}`,
			expectStatus: 400,
		},
		{
			name:         "invalid truncated",
			body:         `{"urls":[["http://example.com"`,
			expectStatus: 400,
		},
	}
	for _, tt := range tests {
		req := httptest.NewRequest("POST", "http://example.com", strings.NewReader(tt.body))
		recorder := httptest.NewRecorder()
		m := DecodeJSONBody[payload](payloadKey{})
		m(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			pp, ok := r.Context().Value(payloadKey{}).(*payload)
			if !ok {
				t.Fatalf("[%s] DecodeJSONBody, expected payload, got %v", tt.name, pp)
			}
		}))(recorder, req)
		response := recorder.Result()
		if response.StatusCode != tt.expectStatus {
			t.Fatalf("[%s] DecodeJSONBody, expected status %d, got %d", tt.name, tt.expectStatus, response.StatusCode)
		}
	}
}

func Test413OnDecodeJSONBody(t *testing.T) {
	t.Parallel()
	type payload struct {
		Urls []string `json:"urls"`
	}
	type payloadKey struct{}
	req := httptest.NewRequest("POST", "http://example.com", strings.NewReader(`{"urls":["http://example.com"]}`))
	w := httptest.NewRecorder()
	m := Chain(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}), MaxBytes(1), DecodeJSONBody[payload](payloadKey{}))
	m(w, req)
	if w.Code != http.StatusRequestEntityTooLarge {
		t.Fatalf("expected status %d, got %d", http.StatusRequestEntityTooLarge, w.Code)
	}
}
