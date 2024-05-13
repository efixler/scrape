package server

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
		m := DecodeJSONBody[payload]()
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
	req := httptest.NewRequest("POST", "http://example.com", strings.NewReader(`{"urls":["http://example.com"]}`))
	w := httptest.NewRecorder()
	m := Chain(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}), MaxBytes(1), DecodeJSONBody[payload]())
	m(w, req)
	if w.Code != http.StatusRequestEntityTooLarge {
		t.Fatalf("expected status %d, got %d", http.StatusRequestEntityTooLarge, w.Code)
	}
}

func TestParseSingleGet(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name         string
		url          string
		expectStatus int
	}{
		{
			name:         "valid",
			url:          "http://example.com",
			expectStatus: 200,
		},
		{
			name:         "invalid",
			url:          "",
			expectStatus: 400,
		},
	}
	for _, tt := range tests {
		req := httptest.NewRequest("GET", "http://example.com?url="+tt.url, nil)
		recorder := httptest.NewRecorder()
		m := parseSinglePayload()
		m(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			pp, ok := r.Context().Value(payloadKey{}).(*singleURLRequest)
			if !ok {
				t.Fatalf("[%s] ParseSingle, expected payload, got %v", tt.name, pp)
			}
			if pp.URL.String() != tt.url {
				t.Errorf("[%s] ParseSingle, expected URL %s, got %s", tt.name, tt.url, pp.URL.String())
			}
		}))(recorder, req)
		response := recorder.Result()
		if response.StatusCode != tt.expectStatus {
			t.Fatalf("[%s] ParseSingle, expected status %d, got %d", tt.name, tt.expectStatus, response.StatusCode)
		}
	}
}

func TestParseSingleJSON(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name         string
		body         string
		expectStatus int
	}{
		{
			name:         "valid",
			body:         `{"url":"http://example.com"}`,
			expectStatus: 200,
		},
		{
			name:         "invalid",
			body:         `{"urls":["http://example.com"]}`,
			expectStatus: 400,
		},
	}
	for _, tt := range tests {
		req := httptest.NewRequest("POST", "http://example.com", strings.NewReader(tt.body))
		recorder := httptest.NewRecorder()
		m := parseSinglePayload()
		m(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			pp, ok := r.Context().Value(payloadKey{}).(*singleURLRequest)
			if !ok {
				t.Fatalf("[%s] ParseSingle, expected payload, got %v", tt.name, pp)
			}
		}))(recorder, req)
		response := recorder.Result()
		if response.StatusCode != tt.expectStatus {
			t.Fatalf("[%s] ParseSingle, expected status %d, got %d", tt.name, tt.expectStatus, response.StatusCode)
		}
	}
}

func TestParseSinglePostForm(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name         string
		body         string
		expectStatus int
	}{
		{
			name:         "valid",
			body:         "url=http://example.com",
			expectStatus: 200,
		},
		{
			name:         "invalid",
			body:         "urls=http://example.com",
			expectStatus: 400,
		},
	}
	for _, tt := range tests {
		req := httptest.NewRequest("POST", "http://example.com", strings.NewReader(tt.body))
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		recorder := httptest.NewRecorder()
		m := parseSinglePayload()
		m(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			pp, ok := r.Context().Value(payloadKey{}).(*singleURLRequest)
			if !ok {
				t.Fatalf("[%s] ParseSingle, expected payload, got %v", tt.name, pp)
			}
		}))(recorder, req)
		response := recorder.Result()
		if response.StatusCode != tt.expectStatus {
			t.Fatalf("[%s] ParseSingle, expected status %d, got %d", tt.name, tt.expectStatus, response.StatusCode)
		}
	}
}

func TestIsJson(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name     string
		content  string
		expected bool
	}{
		{
			name:     "json",
			content:  "application/json",
			expected: true,
		},
		{
			name:     "json with charset",
			content:  "application/json; charset=utf-8",
			expected: true,
		},
		{
			name:     "json with other",
			content:  "application/json; foo=bar",
			expected: true,
		},
		{
			name:     "x-www-form-urlencoded, no addons",
			content:  "application/x-www-form-urlencoded",
			expected: false,
		},
		{
			name:     "x-www-form-urlencoded, with charset",
			content:  "application/x-www-form-urlencoded; charset=utf-8",
			expected: false,
		},
		{
			name:     "multipart, no addons",
			content:  "multipart/form-data",
			expected: false,
		},
		{
			name:     "multipart, with charset",
			content:  "multipart/form-data; charset=utf-8",
			expected: false,
		},
	}
	for _, tt := range tests {
		req := httptest.NewRequest("POST", "http://example.com", nil)
		req.Header.Set("Content-Type", tt.content)
		if isJSON(req) != tt.expected {
			t.Fatalf("[%s] isJSON, expected %v, got %v", tt.name, tt.expected, !tt.expected)
		}
	}
}
