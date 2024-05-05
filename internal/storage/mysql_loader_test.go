//go:build mysql

package storage

import (
	"bytes"
	"context"
	_ "embed"
	"fmt"
	"testing"
	"text/template"

	"github.com/efixler/scrape/database"
	_ "github.com/go-sql-driver/mysql"
)

const (
	testSchema = "scrape_test"
	dbURL      = "root:@tcp(127.0.0.1:3306)/?collation=utf8mb4_0900_ai_ci&multiStatements=true&parseTime=true&readTimeout=30s&timeout=10s&writeTimeout=30s&autocommit=1;"
)

//go:embed mysql/create.sql
var createSQL string

type mysqlConfig struct {
	TargetSchema string
}

var (
	createTemplate = template.Must(template.New("create").Parse(createSQL))
	dbConfig       = mysqlConfig{TargetSchema: testSchema}
)

// Returns a new SQLStorage instance for testing. Each instance returns
// a freshly created db. Since a 'USE' statement is included in the create.sql
// subsequent queries will continue to use the test database.
func getTestDatabase(t *testing.T) *SQLStorage {
	db := New(database.MySQL, dsn)
	err := db.Open(context.TODO())
	if err != nil {
		t.Fatalf("Error opening database: %v", err)
	}
	t.Cleanup(func() {
		if err := db.Close(); err != nil {
			t.Errorf("Error closing mysql database: %v", err)
		}
	})
	var buf bytes.Buffer
	if err = createTemplate.Execute(&buf, dbConfig); err != nil {
		t.Fatalf("Error generating database create sql: %v", err)
	}
	_, err = db.DB.Exec(buf.String())
	if err != nil {
		t.Fatalf("Error creating database: %v", err)
	}
	t.Cleanup(func() {
		q := fmt.Sprintf("DROP DATABASE %v;", dbConfig.TargetSchema)
		if _, err := db.DB.Exec(q); err != nil {
			t.Logf("error dropping mysql test database %q: %v", dbConfig.TargetSchema, err)
		}

	})
	return db
}
