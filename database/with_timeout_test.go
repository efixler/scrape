package database

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"testing"
	"time"
)

func createTempTable(dbh *materialDB, rows int) error {
	create := `CREATE TABLE IF NOT EXISTS test_table (id INTEGER PRIMARY KEY, name TEXT)`
	_, err := dbh.DB.Exec(create)
	if err != nil {
		return err
	}
	insert := `INSERT INTO test_table (id, name) VALUES (?, ?)`
	stmt, err := dbh.DB.Prepare(insert)
	if err != nil {
		return err
	}
	for i := range rows {
		_, err = stmt.Exec(i, randomString(255))
		if err != nil {
			return err
		}
	}
	return nil
}

func randomString(len int) string {
	b := make([]byte, len)
	_, err := rand.Read(b)
	if err != nil {
		panic("Error generating random string")
	}
	return base64.StdEncoding.EncodeToString(b)
}

func TestExecTimeout(t *testing.T) {
	dsn := inMemoryDSN
	dbh := newDB(SQLite, dsn)
	ctx := context.TODO()
	err := dbh.Open(ctx)
	if err != nil {
		t.Fatalf("Error opening database: %s", err)
	}
	defer dbh.Close()
	err = dbh.DB.Ping()
	if err != nil {
		t.Fatalf("Error pinging database: %s", err)
	}
	query := `WITH RECURSIVE r(i) AS (
		VALUES(0)
		UNION ALL
		SELECT i FROM r
		LIMIT 10000000000
	  )
	  SELECT i FROM r WHERE i = 1;`
	_, err = dbh.ExecTimeout(100*time.Millisecond, query)
	if err == nil {
		t.Errorf("Expected error from ExecTimeout, didn't get one")
	} else if err != context.DeadlineExceeded {
		t.Errorf("Expected context.DeadlineExceeded, got %s", err)
	}
	_, err = dbh.Exec(query)
	if err == nil {
		t.Errorf("Expected error from Exec, didn't get one")
	} else if err != context.DeadlineExceeded {
		t.Errorf("Expected context.DeadlineExceeded, got %s", err)
	}

	// In the quert case, we won't see the timeout until we scan the rows
	rows, err := dbh.QueryTimeout(100*time.Millisecond, query)
	if err == nil {
		defer rows.Close()
		rows.Next()
		var ts int
		err = rows.Scan(&ts)
	}
	if err == nil {
		t.Errorf("Expected error from Query, didn't get one")
	} else if err != context.DeadlineExceeded {
		t.Errorf("Expected context.DeadlineExceeded, got %s", err)
	}

	row := dbh.QueryRowTimeout(100*time.Millisecond, query)
	err = row.Err()
	if err == nil {
		var ts int
		err = row.Scan(&ts)
	}
	if err == nil {
		t.Errorf("Expected error from QueryRowTimeout, didn't get one")
	} else if err != context.DeadlineExceeded {
		t.Errorf("Expected context.DeadlineExceeded, got %s", err)
	}
}

func TestQuery(t *testing.T) {
	dsn := inMemoryDSN
	dbh := newDB(SQLite, dsn)
	ctx := context.Background()
	err := dbh.Open(ctx)
	if err != nil {
		t.Fatalf("Error opening database: %s", err)
	}
	defer dbh.Close()
	err = createTempTable(dbh, 100)
	if err != nil {
		t.Fatalf("Error creating temp table: %s", err)
	}
	err = dbh.DB.Ping()
	if err != nil {
		t.Fatalf("Error pinging database: %s", err)
	}
	query := `SELECT * FROM test_table order by id;`
	rows, err := dbh.Query(query)
	if err != nil {
		t.Fatalf("Error from Query: %s", err)
	}
	defer rows.Close()
	count := 0
	for rows.Next() {
		count++
	}
	if count != 100 {
		t.Errorf("Expected 100 rows, got %d", count)
	}
	query = `SELECT name FROM test_table WHERE id = 0;`
	row := dbh.QueryRow(query)
	err = row.Err()
	if err != nil {
		t.Fatalf("Error from QueryRow: %s", err)
	}
	var name string
	err = row.Scan(&name)
	if err != nil {
		t.Errorf("Row scan failed %s", err)
	}
}
