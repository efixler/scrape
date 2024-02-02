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

func Factory(opts Options) store.Factory {
	return func() (store.URLDataStore, error) {
		store := &MySQLStore{
			DBHandle: database.DBHandle[stmtIndex]{
				Driver:    database.MySQL,
				DSNSource: opts,
			},
		}
		return store, nil
	}
}

func New(opts Options) (store.URLDataStore, error) {
	return Factory(opts)()
}

type MySQLStore struct {
	database.DBHandle[stmtIndex]
}

func (s *MySQLStore) Fetch(*nurl.URL) (*resource.WebPage, error) {
	return nil, nil
}

func (s *MySQLStore) Store(*resource.WebPage) (uint64, error) {
	return 0, nil
}
