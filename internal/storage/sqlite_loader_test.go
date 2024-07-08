//go:build !mysql

package storage

import (
	"github.com/efixler/scrape/database"
	"github.com/efixler/scrape/database/sqlite"
	_ "github.com/mattn/go-sqlite3"
)

func getTestDatabaseEngine() database.Engine {
	return sqlite.MustNew(
		sqlite.InMemoryDB(),
	)
}
