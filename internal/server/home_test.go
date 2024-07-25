package server

import (
	"bytes"
	"context"
	"net/http/httptest"
	"testing"

	"github.com/efixler/scrape/internal/auth"
	"golang.org/x/net/html"
)

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
		as := newAdminServer()

		ss := MustScrapeServer(
			context.Background(),
			WithURLFetcher(&mockUrlFetcher{}),
			WithAuthorizationIf(test.key),
		)
		tmpl := as.mustHomeTemplate(ss, test.openHome)
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

func TestMustBaseTemplate(t *testing.T) {
	as := newAdminServer()
	tmpl := as.mustBaseTemplate()
	if tmpl == nil {
		t.Fatal("Expected non-nil template")
	}
	requiredTemplates := map[string]bool{
		"base.html":    false,
		"menubar.html": false,
		// following are blocks expected to be defined
		"content": false,
		"head":    false,
		"scripts": false,
		"title":   false,
	}
	for _, t := range tmpl.Templates() {
		requiredTemplates[t.Name()] = true
	}
	for k, v := range requiredTemplates {
		if !v {
			t.Errorf("Expected template %s to be defined", k)
		}
	}
	if tmpl == as.baseTemplate {
		t.Error("Expected returned base template to be a clone baseTemplate")
	}
}

func TestSettingsHandler(t *testing.T) {
	as := newAdminServer()
	handler := as.settingsHandler()
	if handler == nil {
		t.Fatal("Expected non-nil handler")
	}
	req := httptest.NewRequest("GET", "http://foo.bar/", nil)
	w := httptest.NewRecorder()
	handler(w, req)
	resp := w.Result()
	if resp.StatusCode != 200 {
		t.Errorf("Expected 200 status code, got %d", resp.StatusCode)
	}
	if _, err := html.Parse(resp.Body); err != nil {
		t.Errorf("Error parsing settings rendered content body: %s", err)
	}
}
