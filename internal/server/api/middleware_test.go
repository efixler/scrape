package api

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

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
			pp, ok := r.Context().Value(payloadKey{}).(*SingleURLRequest)
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
			pp, ok := r.Context().Value(payloadKey{}).(*SingleURLRequest)
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
			pp, ok := r.Context().Value(payloadKey{}).(*SingleURLRequest)
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
