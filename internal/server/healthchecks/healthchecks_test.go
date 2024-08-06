package healthchecks

import (
	"encoding/json"
	"net/http/httptest"
	"path"
	"strings"
	"testing"
)

func TestHeartbeat(t *testing.T) {
	type data struct {
		root string
	}
	tests := []data{
		{""},
		{"/"},
		{"/.well-known"},
		{"/.well-known/"},
	}

	testF := func(root string) {
		ts := httptest.NewServer(Handler(root, nil))
		defer ts.Close()
		client := ts.Client()
		urlPath := path.Clean(root + "/heartbeat")
		targetUrl := ts.URL + urlPath
		resp, err := client.Get(targetUrl)
		if err != nil {
			t.Fatal(err)
		}
		if resp.StatusCode != 200 {
			t.Errorf("Expected 200 OK, got %d (root: %s, url: %s)", resp.StatusCode, root, targetUrl)
		}
	}

	for _, test := range tests {
		testF(test.root)
	}
}

func TestHealthHandler(t *testing.T) {
	t.Parallel()
	type data struct {
		root string
	}
	tests := []data{
		{""},
		{"/"},
		{"/.well-known"},
		{"/.well-known/"},
	}

	testF := func(root string) {
		ts := httptest.NewServer(Handler(root, nil))
		defer ts.Close()
		client := ts.Client()
		urlPath := path.Clean(root + "/health")
		targetUrl := ts.URL + urlPath
		resp, err := client.Get(targetUrl)
		if err != nil {
			t.Fatal(err)
		}
		if resp.StatusCode != 200 {
			t.Errorf("Expected 200 OK, got %d (root: %s, url: %s)", resp.StatusCode, root, targetUrl)
		}
		cType := resp.Header.Get("Content-Type")
		if !strings.HasPrefix(cType, "application/json") {
			t.Errorf("Expected JSON content type, got %q", cType)
		}

		var h health
		decoder := json.NewDecoder(resp.Body)
		err = decoder.Decode(&h)
		if err != nil {
			t.Fatal(err)
		}
		app := h.Application
		if app.StartTime == "" {
			t.Errorf("Expected non-empty start time, got %q", app.StartTime)
		}
		if app.GoroutineCount == 0 {
			t.Errorf("Expected non-zero goroutine count, got %d", app.GoroutineCount)
		}
		if h.Memory == nil {
			t.Errorf("Expected non-nil memory, got nil")
		}
		if h.Memory.System == 0 {
			t.Errorf("Expected non-zero system memory, got %d", h.Memory.System)
		}
		if h.Memory.HeapSys == 0 {
			t.Errorf("Expected non-zero heap memory, got %d", h.Memory.HeapSys)
		}
	}

	for _, test := range tests {
		testF(test.root)
	}
}
