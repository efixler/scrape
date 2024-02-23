//go:build mysql

package storage

import (
	"bytes"
	"context"
	_ "embed"
	"testing"
	"text/template"

	"github.com/efixler/scrape/database"
	_ "github.com/go-sql-driver/mysql"
)

const (
	testSchema = "scrape_test"
	dbURL      = "root:@tcp(127.0.0.1:3306)/?collation=utf8mb4_0900_ai_ci&multiStatements=true&parseTime=true&readTimeout=30s&timeout=10s&writeTimeout=30s"
)

//go:embed mysql/create.sql
var createSQL string

type mysqlConfig struct {
	DBName string
}

var (
	createTemplate = template.Must(template.New("create").Parse(createSQL))
	dbConfig       = mysqlConfig{DBName: testSchema}
)

// Returns a new SQLStorage instance for testing. Each instance returns
// a freshly created db. Since a 'USE' statement is included in the create.sql
// subsequent queries will continue to use the test database.
func db(t *testing.T) *SQLStorage {
	db := New(database.MySQL)
	db.DSNSource = dsn
	err := db.Open(context.TODO())
	if err != nil {
		t.Fatalf("Error opening database: %v", err)
	}
	var buf bytes.Buffer
	if err = createTemplate.Execute(&buf, dbConfig); err != nil {
		t.Fatalf("Error generating database create sql: %v", err)
	}
	_, err = db.DB.Exec(buf.String())
	if err != nil {
		t.Fatalf("Error creating database: %v", err)
	}
	return db
}
