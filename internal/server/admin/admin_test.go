package admin

import (
	"net/http/httptest"
	"testing"

	"github.com/efixler/scrape/internal/auth"
	"golang.org/x/net/html"
)

func TestMustBaseTemplate(t *testing.T) {
	as := MustServer(nil)
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
	as := MustServer(nil)
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

func TestWithBasePathOption(t *testing.T) {
	tests := []struct {
		name      string
		basePath  string
		expectErr bool
	}{
		{
			name:      "empty base path",
			basePath:  "",
			expectErr: true,
		},
		{
			name:      "valid base path",
			basePath:  "/foo",
			expectErr: false,
		},
	}
	for _, test := range tests {
		c := &config{}
		err := WithBasePath(test.basePath)(c)
		if test.expectErr && err == nil {
			t.Errorf("[%s] Expected error, got nil", test.name)
		}
		if !test.expectErr && err != nil {
			t.Errorf("[%s] Expected no error, got %s", test.name, err)
		}
		if c.basePath != test.basePath {
			t.Errorf("[%s] Expected base path %s, got %s", test.name, test.basePath, c.basePath)
		}
	}
}

func TestWithAuthzOption(t *testing.T) {
	tests := []struct {
		name          string
		authz         AuthzProvider
		expectEnabled bool
	}{
		{
			name:          "nil authz",
			authz:         nil,
			expectEnabled: false,
		},
		{
			name:          "non-nil no authz",
			authz:         authzShim{},
			expectEnabled: false,
		},
		{
			name:          "non-nil authz",
			authz:         authzShim(auth.MustHS256SigningKey()),
			expectEnabled: true,
		},
	}
	for _, test := range tests {
		c := &config{}
		err := WithAuthz(test.authz)(c)
		if err != nil {
			t.Errorf("[%s] Unexpected error: %s", test.name, err)
		}
		if c.authz == nil {
			t.Errorf("[%s] Expected non-nil authz", test.name)
		}
		if c.authz.AuthEnabled() != test.expectEnabled {
			t.Errorf("[%s] Expected auth enabled %t, got %t", test.name, test.expectEnabled, c.authz.AuthEnabled())
		}
	}
}
