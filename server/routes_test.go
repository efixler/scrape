package server

import (
	"encoding/json"
	"fmt"
	"net/http/httptest"
	"slices"
	"testing"
)

func TestMutateFeedRequestForBatch(t *testing.T) {
	type data struct {
		url         string
		expectPP    string
		expectOther map[string]string
	}

	tests := []data{
		{"https://foo.com?pp=1&url=http://foo.bar&crunk=X", "1", map[string]string{"crunk": ""}},
	}

	for _, test := range tests {
		var request = httptest.NewRequest("GET", test.url, nil)
		var urls = []string{
			"https://arstechnica.com/?p=1993801",
			"https://arstechnica.com/?p=1993618",
			"https://arstechnica.com/?p=1993507",
			"https://arstechnica.com/?p=1993162",
		}
		mutated := mutateFeedRequestForBatch(request, urls)
		if mutated.Header.Get("Content-Type") != "application/json" {
			t.Errorf("Expected Content-Type 'application/json', got '%s'", mutated.Header.Get("Content-Type"))
		}
		decoder := json.NewDecoder(mutated.Body)
		var batchRequest BatchRequest
		err := decoder.Decode(&batchRequest)
		if err != nil {
			t.Errorf("Error decoding JSON: %s", err)
		}
		fmt.Printf("Form Values %v", mutated.Form)
		if len(batchRequest.Urls) != len(urls) {
			t.Errorf("Expected %d urls, got %d", len(urls), len(batchRequest.Urls))
		}
		if !slices.Equal(batchRequest.Urls, urls) {
			t.Errorf("Expected %v, got %v", urls, batchRequest.Urls)
		}
		if mutated.FormValue("pp") != test.expectPP {
			t.Errorf("Expected PrettyPrint %v, got %v", request.FormValue("pp"), mutated.FormValue("pp"))
		}
		if mutated.FormValue("url") != "" {
			t.Errorf("Expected url to be empty, got %v", mutated.FormValue("url"))
		}
		for k, v := range test.expectOther {
			if mutated.FormValue(k) != v {
				t.Errorf("Expected %s=%s, got %s=%s", k, v, k, mutated.FormValue(k))
			}
		}
	}

}
