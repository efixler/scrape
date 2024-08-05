package middleware

import (
	"net/http/httptest"
	"testing"
)

func TestIsJson(t *testing.T) {
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
		{
			name:     "empty",
			content:  "",
			expected: true,
		},
	}
	for _, tt := range tests {
		req := httptest.NewRequest("POST", "http://example.com", nil)
		req.Header.Set("Content-Type", tt.content)
		if IsJSONRequest(req) != tt.expected {
			t.Fatalf("[%s] isJSON, expected %v, got %v", tt.name, tt.expected, !tt.expected)
		}
	}
}
