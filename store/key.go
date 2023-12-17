// Key generation and related methods relevant to any storage backend.
package store

import (
	"fmt"
	"hash/fnv"
	"regexp"
)

// net.URL and URLString both implement this interface, which is needed
// to generate a key for the URL.
type URLWithHostname interface {
	fmt.Stringer
	Hostname() string
}

// Type that provides the
type URLString string

func (u URLString) String() string {
	return string(u)
}

var extractHostFromUrl = regexp.MustCompile(`^https?://([^/]+)`)

func (u URLString) Hostname() string {
	matches := extractHostFromUrl.FindStringSubmatch(u.String())
	if len(matches) < 2 {
		return ""
	}
	return matches[1]
}

const (
	MASK_56 uint64 = 0xffffffffffffff

	CHECKSUM_MASK uint64 = 127 << 56
	TIME_MASK     uint64 = 0xFFFFFF << 32
)

// Produces a 63 bit uint contained in a uint64
// (SQLite cannot accept uint64 with high bit set as a primary key)
// [Bit 63] Always 0
// [Bits 62-56] A 7 bit checksum based on the domain name
// [Bits 31-0] A 56 bit hash of the URL (reduced from a 64 bit fnv1a hash)
func Key(url URLWithHostname) uint64 {
	dbytes := []byte(url.Hostname())
	var sum uint8
	for _, b := range dbytes {
		sum += b
	}
	seg := (uint64(sum) << 56) & CHECKSUM_MASK

	h := fnv.New64a()
	h.Write([]byte(url.String()))
	hash := uint64(h.Sum64())
	hash = (hash >> 56) ^ (hash & MASK_56)
	return seg | hash
}
