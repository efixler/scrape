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
		if err == nil && c.host != test.host {
			t.Errorf("Host: unexpected host: %s", c.host)
		}
		if err != test.expectedErr {
			t.Errorf("%s: unexpected error: %s", test.name, err)
		}
	}
}

func TestPort(t *testing.T) {
	t.Parallel()
	type data struct {
		name        string
		port        int
		expectedErr error
	}
	tests := []data{
		{"zero", 0, store.ErrorValueNotAllowed},
		{"negative", -1, store.ErrorValueNotAllowed},
		{"positive", 1, nil},
	}
	for _, test := range tests {
		c := defaultConfig()
		err := Port(test.port)(&c)
		if err != test.expectedErr {
			t.Errorf("%s: unexpected error: %s", test.name, err)
		}
		if err == nil && c.port != test.port {
			t.Errorf("Port - %s: unexpected port: %d", test.name, c.port)
		}

	}
}

func TestUsername(t *testing.T) {
	t.Parallel()
	type data struct {
		name        string
		username    string
		expectedErr error
	}
	tests := []data{
		{"empty", "", store.ErrorValueNotAllowed},
		{"foo", "foo", nil},
	}
	for _, test := range tests {
		c := defaultConfig()
		err := Username(test.username)(&c)
		if err == nil && c.username != test.username {
			t.Errorf("Username - %s: unexpected username: %s", test.name, c.username)
		}
		if err != test.expectedErr {
			t.Errorf("%s: unexpected error: %s", test.name, err)
		}
	}
}

func TestPassword(t *testing.T) {
	t.Parallel()
	type data struct {
		name        string
		password    string
		expectedErr error
	}
	tests := []data{
		{"empty", "", nil},
		{"not empty", "foo", nil},
	}
	for _, test := range tests {
		c := defaultConfig()
		err := Password(test.password)(&c)
		if err == nil && c.password != test.password {
			t.Errorf("%s: unexpected password: %s", test.name, c.password)
		}
		if err != test.expectedErr {
			t.Errorf("%s: unexpected error: %s", test.name, err)
		}
	}
}

func TestAddress(t *testing.T) {
	t.Parallel()
	type data struct {
		name        string
		address     string
		expectHost  string
		expectPort  int
		expectError bool
	}
	tests := []data{
		{"empty", "", "", 0, true},
		{"localhost", "localhost", "localhost", 3306, false},
		{"localhost with port", "localhost:5000", "localhost", 5000, false},
		{"localhost with invalid port", "localhost:foo", "", 0, true},
		{"127.0.0.1", "127.0.0.1", "127.0.0.1", 3306, false},
		{"127 with port", "127.0.0.1:5000", "127.0.0.1", 5000, false},
	}
	for _, test := range tests {
		c := defaultConfig()
		err := Address(test.address)(&c)
		if (err != nil) != test.expectError {
			t.Fatalf("%s: unexpected error: %s", test.name, err)
		} else if test.expectError {
			continue
		}
		if c.host != test.expectHost {
			t.Errorf("%s: unexpected host: %q, expected %q", test.name, c.host, test.expectHost)
		}
		if c.port != test.expectPort {
			t.Errorf("%s: unexpected port: %d, expected %d", test.name, c.port, test.expectPort)
		}
	}
}
