//go:build mysql

package settings

import (
	"github.com/efixler/scrape/database"
	"github.com/efixler/scrape/database/mysql"
)

func testEngine() database.Engine {
	engine := mysql.MustNew(
		mysql.NetAddress("127.0.0.1:3306"),
		mysql.Username("root"),
		mysql.WithMaxConnections(1),
		mysql.Schema("scrape_test"),
		mysql.ForMigration(),
	)
	return engine
}
