package server

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestUnmarshalSingleUrlRequest(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name            string
		body            string
		expectURLString string
		expectPP        bool
		expectErr       bool
	}{
		{
			name:            "valid",
			body:            `{"url":"http://example.com"}`,
			expectURLString: "http://example.com",
			expectPP:        false,
			expectErr:       false,
		},
		{
			name:            "missing url",
			body:            `{"urls":["http://example.com"]}`,
			expectURLString: "",
			expectPP:        false,
			expectErr:       true,
		},
		{
			name:            "non-absolute url",
			body:            `{"url":"example/foo"}`,
			expectURLString: "",
			expectPP:        false,
			expectErr:       true,
		},
		{
			name:            "url with fragment",
			body:            `{"url":"http://example.com#fragment"}`,
			expectURLString: "http://example.com#fragment",
			expectPP:        false,
			expectErr:       false,
		},
	}
	for _, tt := range tests {
		var sur singleURLRequest
		err := sur.UnmarshalJSON([]byte(tt.body))
		if (err != nil) != tt.expectErr {
			t.Fatalf("[%s] UnmarshalSingleUrlRequest, expected error %v, got %v", tt.name, tt.expectErr, err)
		}
		if tt.expectURLString != "" && sur.URL.String() != tt.expectURLString {
			t.Errorf("[%s] UnmarshalSingleUrlRequest, expected URL %s, got %s", tt.name, tt.expectURLString, sur.URL.String())
		}
		if sur.PrettyPrint != tt.expectPP {
			t.Errorf("[%s] UnmarshalSingleUrlRequest, expected PrettyPrint %v, got %v", tt.name, tt.expectPP, sur.PrettyPrint)
		}
		// now run the same test but with json.Decoder
		reader := strings.NewReader(tt.body)
		decoder := json.NewDecoder(reader)
		decoder.DisallowUnknownFields()
		surD := new(singleURLRequest)
		err = decoder.Decode(surD)
		if (err != nil) != tt.expectErr {
			t.Fatalf("[%s] json.Decoder.Decode, expected error %v, got %v", tt.name, tt.expectErr, err)
		}
		if tt.expectURLString != "" && surD.URL.String() != tt.expectURLString {
			t.Errorf("[%s] json.Decoder.Decode, expected URL %s, got %s", tt.name, tt.expectURLString, surD.URL.String())
		}
		if surD.PrettyPrint != tt.expectPP {
			t.Errorf("[%s] json.Decoder.Decode, expected PrettyPrint %v, got %v", tt.name, tt.expectPP, surD.PrettyPrint)
		}
	}
}
