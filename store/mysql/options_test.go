package mysql

import (
	"testing"
	"time"

	"github.com/efixler/scrape/store"
)

func TestDSN(t *testing.T) {
	t.Parallel()
	o := defaultConfig()
	o.host = "localhost"
	o.username = "user"
	o.password = "password"
	o.database = "test"
	o.port = 5000
	o.timeout = 5 * time.Second
	o.parseTime = true
	if o.DSN() != "user:password@tcp(localhost:5000)/test?charset=utf8mb4&parseTime=true&loc=UTC&timeout=5s" {
		t.Errorf("DSN: unexpected DSN: %s", o.DSN())
	}
	if o.String() != "user:*****@tcp(localhost:5000)/test?charset=utf8mb4&parseTime=true&loc=UTC&timeout=5s" {
		t.Errorf("String: unexpected DSN: %s", o.String())
	}
}

func TestHost(t *testing.T) {
	t.Parallel()
	type data struct {
		name        string
		host        string
		expectedErr error
	}
	tests := []data{
		{"empty", "", store.ErrorValueNotAllowed},
		{"localhost", "localhost", nil},
		{"127.0.0.1", "127.0.0.1", nil},
	}
	for _, test := range tests {
		c := defaultConfig()
		err := Host(test.host)(&c)
		if c.host != test.host {
			t.Errorf("Host: unexpected host: %s", c.host)
		}
		if err != test.expectedErr {
			t.Errorf("unexpected error: %s", err)
		}
	}
}
