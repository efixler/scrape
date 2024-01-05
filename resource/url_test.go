package resource

import (
	nurl "net/url"
	"testing"
)

func TestCleanURL(t *testing.T) {
	type data struct {
		url      string
		expected string
	}
	tests := []data{
		{"https://example.com", "https://example.com"},
		{"https://example.com?utm_source=foo", "https://example.com"},
		{"https://example.com?utm_source=foo&utm_medium=bar", "https://example.com"},
		{"https://example.com?utm_source=foo&utm_medium=bar&utm_campaign=baz", "https://example.com"},
		{"https://example.com?utm_source=foo&utm_medium=bar&utm_campaign=baz&utm_term=quux", "https://example.com"},
		{"https://example.com?utm_source=foo&utm_medium=bar&utm_campaign=baz&utm_term=quux&utm_content=xyzzy", "https://example.com"},
		{"https://example.com?utm_source=foo&utm_medium=bar&utm_campaign=baz&utm_term=quux&utm_content=xyzzy&foo=bar", "https://example.com?foo=bar"},
		{"https://example.com?utm_source=foo&utm_medium=bar&utm_campaign=baz&utm_term=quux&utm_content=xyzzy&foo=bar&baz=quux", "https://example.com?baz=quux&foo=bar"},
	}
	for _, test := range tests {
		url, _ := nurl.Parse(test.url)
		cleaned := CleanURL(url)
		if cleaned.String() != test.expected {
			t.Errorf("Expected %s, got %s", test.expected, cleaned.String())
		}
	}
}
