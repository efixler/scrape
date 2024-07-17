package server

import (
	"bytes"
	"context"
	"testing"

	"github.com/efixler/scrape/internal/auth"
)

func TestMustTemplate(t *testing.T) {
	tests := []struct {
		name        string
		key         auth.HMACBase64Key
		expectToken bool
	}{
		{
			name:        "with key",
			key:         auth.MustNewHS256SigningKey(),
			expectToken: true,
		},
		{
			name:        "no key",
			key:         nil,
			expectToken: false,
		},
		{
			name:        "empty key",
			key:         auth.HMACBase64Key([]byte{}),
			expectToken: false,
		},
	}
	for _, test := range tests {
		ss := MustScrapeServer(
			context.Background(),
			WithURLFetcher(&mockUrlFetcher{}),
			WithAuthorizationIf(test.key),
		)
		tmpl := mustHomeTemplate(ss)
		tmpl, err := tmpl.Parse("{{AuthToken}}")
		if err != nil {
			t.Fatalf("[%s] Error parsing template: %s", test.name, err)
		}
		var buf bytes.Buffer
		err = tmpl.Execute(&buf, nil)
		if err != nil {
			t.Fatalf("[%s] Error executing template: %s", test.name, err)
		}
		output := buf.String()
		if !test.expectToken && output != "" {
			t.Fatalf("[%s] Expected empty output, got %s", test.name, output)
		}
		if test.expectToken {
			switch output {
			case "":
				t.Fatalf("[%s] Expected non-empty token, got empty", test.name)
			default:
				_, err := auth.VerifyToken(test.key, output)
				if err != nil {
					t.Fatalf("[%s] Error verifying token: %s", test.name, err)
				}
			}
		}
	}
}
