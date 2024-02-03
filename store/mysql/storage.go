package mysql

import (
	nurl "net/url"

	_ "github.com/go-sql-driver/mysql"

	"github.com/efixler/scrape/resource"
	"github.com/efixler/scrape/store"
	"github.com/efixler/scrape/store/internal/database"
)

type stmtIndex int

const (
	_ stmtIndex = iota
)

func Factory(options ...option) store.Factory {
	return func() (store.URLDataStore, error) {
		return New(options...)
	}
}

func New(options ...option) (store.URLDataStore, error) {
	store := &Store{
		DBHandle: database.DBHandle[stmtIndex]{
			Driver: database.MySQL,
		},
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
	database.DBHandle[stmtIndex]
}

func (s *Store) Fetch(*nurl.URL) (*resource.WebPage, error) {
	return nil, nil
}

func (s *Store) Store(*resource.WebPage) (uint64, error) {
	return 0, nil
}
