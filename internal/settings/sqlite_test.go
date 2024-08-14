//go:build !mysql

package settings

import (
	"github.com/efixler/scrape/database"
	"github.com/efixler/scrape/database/sqlite"
)

func testEngine() database.Engine {
	engine := sqlite.MustNew(sqlite.InMemoryDB())
	return engine
}
