package settings

import (
	"testing"
)

func TestMIMEHeaderMarshalJSON(t *testing.T) {
	mh := make(MIMEHeader)
	mh["content-type"] = "text/html"
	mh["Content-Length"] = "1024"
	json, err := mh.MarshalJSON()
	if err != nil {
		t.Errorf("MarshalJSON() failed: %v", err)
	}
	expected := `{"Content-Length":"1024","Content-Type":"text/html"}`
	if string(json) != expected {
		t.Errorf("MarshalJSON() = %v, want %v", string(json), expected)
	}
}
