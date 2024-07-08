//go:build mysql

package storage

import (
	_ "embed"

	"github.com/efixler/scrape/database"
	"github.com/efixler/scrape/database/mysql"
	_ "github.com/go-sql-driver/mysql"
)

const (
	testSchema = "scrape_test"
)

func getTestDatabaseEngine() database.Engine {
	return mysql.MustNew(
		mysql.NetAddress("127.0.0.1:3306"),
		mysql.Username("root"),
		mysql.WithMaxConnections(1),
		mysql.Schema(testSchema),
		mysql.ForMigration(),
	)
}
