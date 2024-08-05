package admin

import (
	"bytes"
	"net/http/httptest"
	"testing"

	"github.com/efixler/scrape/internal/auth"
)

// HomeHander is open by intent
func TestHomeHandlerAuth(t *testing.T) {
	tests := []struct {
		name           string
		key            auth.HMACBase64Key
		expectedResult int
	}{
		{
			name:           "no key",
			key:            nil,
			expectedResult: 200,
		},
		{
			name:           "good key",
			key:            auth.MustNewHS256SigningKey(),
			expectedResult: 200,
		},
	}
	for _, test := range tests {
		authz := authzShim(test.key)

		for _, openHome := range []bool{true, false} {
			req := httptest.NewRequest("GET", "http://foo.bar/", nil)
			w := httptest.NewRecorder()
			MustServer(nil, WithAuthz(authz)).homeHandler(openHome)(w, req)
			resp := w.Result()
			if resp.StatusCode != test.expectedResult {
				t.Errorf("[%s] Expected %d, got %d", test.name, test.expectedResult, resp.StatusCode)
			}
		}
	}
}

func TestHomeTemplateAuthSettings(t *testing.T) {
	tests := []struct {
		name        string
		key         auth.HMACBase64Key
		openHome    bool
		expectToken bool
	}{
		{
			name:        "auth enabled",
			key:         auth.MustNewHS256SigningKey(),
			openHome:    false,
			expectToken: false,
		},
		{
			name:        "auth enabled with open home",
			key:         auth.MustNewHS256SigningKey(),
			openHome:    true,
			expectToken: true,
		},
		{
			name:        "auth disabled",
			key:         nil,
			openHome:    false,
			expectToken: false,
		},
		{
			name:        "empty key",
			key:         auth.HMACBase64Key([]byte{}),
			openHome:    false,
			expectToken: false,
		},
	}
	for _, test := range tests {
		as := MustServer(nil)

		authz := authzShim(test.key)

		tmpl := as.mustHomeTemplate(authz, test.openHome)
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
