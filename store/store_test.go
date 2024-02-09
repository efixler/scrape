package store

import (
	_ "embed"
	"encoding/json"
	"testing"

	"github.com/efixler/scrape/resource"
)

//go:embed store_test.json
var testJson []byte

func TestSerializedMetadataFiltersWhenMarshaling(t *testing.T) {
	// We expect filename to be resolved to an absolute path
	saver := &resource.WebPage{}
	err := json.Unmarshal(testJson, saver)
	if err != nil {
		t.Fatalf("Error unmarshaling base metadata: %v", err)
	}
	filtered, err := SerializeMetadata(saver)

	s := &resource.WebPage{}
	err = json.Unmarshal(filtered, s)

	if err != nil {
		t.Fatalf("Error marshaling serialized metadata: %v", err)
	}
	if s.ContentText != "" {
		t.Errorf("Serialized metadata should have filtered out ContentText: %s", s.ContentText)
	}
	if s.FetchTime != nil {
		t.Errorf("Serialized metadata should have filtered out FetchTime: %v", s.FetchTime)
	}
	if s.OriginalURL != "" {
		t.Errorf("Serialized metadata should have filtered out OriginalURL: %s", s.OriginalURL)
	}
	if s.RequestedURL != nil {
		t.Errorf("Serialized metadata should have filtered out RequestedURL: %v", s.RequestedURL)
	}
}
