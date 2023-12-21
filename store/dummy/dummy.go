package store

import "github.com/efixler/scrape/store"

type DummyStore struct {
}

// Generic struct creator with options for different types of stores and store options

func Open(path string, options any) (*store.StoredUrlData, error) {
	return nil, nil
}
