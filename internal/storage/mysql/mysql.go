package mysql

import (
	"github.com/efixler/scrape/database"
	"github.com/efixler/scrape/internal/storage"
	"github.com/efixler/scrape/store"
)

func Factory(options ...Option) store.Factory {
	return func() (store.URLDataStore, error) {
		return New(options...)
	}
}

func New(options ...Option) (store.URLDataStore, error) {
	store := &Store{
		storage.New(database.MySQL),
	}
	config := defaultConfig()
	for _, opt := range options {
		if err := opt(&config); err != nil {
			return nil, err
		}
	}
	store.DSNSource = config
	return store, nil
}

type Store struct {
	*storage.SQLStorage
}
