package healthchecks

import (
	"net/http/httptest"
	"path"
	"testing"
)

func TestHeartbeat(t *testing.T) {
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
		ts := httptest.NewServer(Handler(root))
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
