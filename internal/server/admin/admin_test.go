package admin

import (
	"net/http/httptest"
	"testing"

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
