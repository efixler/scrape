package mysql

import (
	"fmt"
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

func Factory(username, password, host string, port int) store.Factory {
	dsnF := func() string {
		return fmt.Sprintf(
			"%s:%s@tcp(%s:%d)/scrape?charset=utf8mb4&parseTime=True&loc=UTC&timeout=10s",
			username,
			password,
			host,
			port,
		)
	}
	return func() (store.URLDataStore, error) {
		store := &MySQLStore{
			DBHandle: database.DBHandle[stmtIndex]{
				Driver: database.MySQL,
				DSN:    dsnF,
			},
		}
		return store, nil
	}
}

func New(username, password, host string, port int) (store.URLDataStore, error) {
	return Factory(username, password, host, port)()
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
