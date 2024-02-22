//go:build mysql

package storage

import (
	"context"
	_ "embed"
	"testing"

	"github.com/efixler/scrape/database"
	_ "github.com/go-sql-driver/mysql"
)

const (
	dbURL = "root:@tcp(127.0.0.1:3306)/scrape?collation=utf8mb4_0900_ai_ci&multiStatements=true&parseTime=true&readTimeout=30s&timeout=10s&writeTimeout=30s"
)

//go:embed mysql/create.sql
var createSQL string

func db(t *testing.T) *SQLStorage {
	db := New(database.MySQL)
	db.DSNSource = dsn
	err := db.Open(context.TODO())
	if err != nil {
		t.Fatalf("Error opening database: %v", err)
	}
	_, err = db.DB.Exec(createSQL)
	if err != nil {
		t.Fatalf("Error creating database: %v", err)
	}
	return db
}
