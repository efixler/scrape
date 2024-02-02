package mysql

import (
	"testing"
	"time"
)

func TestDSN(t *testing.T) {
	t.Parallel()
	o := DefaultOptions()
	o.Host = "localhost"
	o.Username = "user"
	o.Password = "password"
	o.database = "test"
	o.Port = 5000
	o.timeout = 5 * time.Second
	o.parseTime = true
	if o.DSN() != "user:password@tcp(localhost:5000)/test?charset=utf8mb4&parseTime=true&loc=UTC&timeout=5s" {
		t.Errorf("DSN: unexpected DSN: %s", o.DSN())
	}
	if o.String() != "user:*****@tcp(localhost:5000)/test?charset=utf8mb4&parseTime=true&loc=UTC&timeout=5s" {
		t.Errorf("String: unexpected DSN: %s", o.String())
	}
}
