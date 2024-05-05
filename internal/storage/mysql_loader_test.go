//go:build mysql

package storage

import (
	"bytes"
	"context"
	"embed"
	_ "embed"
	"fmt"
	"testing"
	"text/template"

	"github.com/efixler/scrape/database"
	_ "github.com/go-sql-driver/mysql"
	"github.com/pressly/goose/v3"
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

//go:embed mysql/migrations/*.sql
var migrationsFS embed.FS

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
	goose.SetBaseFS(migrationsFS)
	if err := goose.SetDialect(string(goose.DialectMySQL)); err != nil {
		t.Fatalf("Error setting dialect: %v", err)
	}
	if err := goose.Up(db.DB, "mysql/migrations"); err != nil {
		t.Fatalf("Error creating MySQL test db via migration: %v", err)
	}
	return db
}
