package mysql

import (
	"strconv"
	"strings"
	"testing"

	"github.com/efixler/scrape/store"
)

func TestPasswordMaskingOnString(t *testing.T) {
	t.Parallel()
	c := defaultConfig()
	Username("root")(&c)
	Password("password")(&c)
	NetAddress("localhost")(&c)

	str := c.String()
	if !strings.HasPrefix(str, "root:*****@") {
		t.Errorf("String: unexpected DSN string, password not masked: %s", str)
	}
	dsn := c.DSN()
	if !strings.HasPrefix(dsn, "root:password@") {
		t.Errorf("DSN: unexpected DSN string, password incorrect: %s", dsn)
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
		if err == nil && c.User != test.username {
			t.Errorf("Username - %s: unexpected username: %s", test.name, c.User)
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
		if err == nil && c.Passwd != test.password {
			t.Errorf("%s: unexpected password: %s", test.name, c.Passwd)
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
		err := NetAddress(test.address)(&c)
		if (err != nil) != test.expectError {
			t.Fatalf("%s: unexpected error: %s", test.name, err)
		} else if test.expectError {
			continue
		}
		elems := strings.SplitN(c.Addr, ":", 2)
		if elems[0] != test.expectHost {
			t.Errorf("%s: unexpected host: %q, expected %q", test.name, elems[0], test.expectHost)
		}
		port, err := strconv.Atoi(elems[1])
		if err != nil {
			t.Errorf("%s: non-numeric port: expected %d, got %q, %s", test.name, test.expectPort, elems[1], err)
		}
		if port != test.expectPort {
			t.Errorf("%s: unexpected port: %d, expected %d", test.name, port, test.expectPort)
		}
	}
}

func TestWithoutSchema(t *testing.T) {
	t.Parallel()
	c := defaultConfig()
	WithoutSchema()(&c)
	if c.DBName != "" {
		t.Errorf("WithoutSchema: unexpected schema: %s", c.DBName)
	}
}
