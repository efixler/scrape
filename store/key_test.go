package store

import (
	"testing"
)

func TestUnsetTopBit(t *testing.T) {
	dummyDomain := string([]byte{255})
	url := URLString("http://" + dummyDomain + "/foo/bar")
	key := Key(url)
	negator := uint64(1 << 63)
	if key&negator != 0 {
		t.Errorf("Key(%s) = %b; expected %b", url, key, key&negator)
	}
}
