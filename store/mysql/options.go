package mysql

import (
	"fmt"
	"time"

	"github.com/efixler/scrape/store"
)

type Charset string
type Location string

const (
	Utf8mb4     Charset  = "utf8mb4"
	UTC         Location = "UTC"
	DefaultPort          = 3306
	dsnFmt               = "%s:%s@tcp(%s:%d)/%s?charset=%s&parseTime=%t&loc=%s&timeout=%s"
	dbSchema             = "scrape"
)

var (
	DefaultTimeout = 10 * time.Second
)

type option func(*Config) error

func Host(host string) option {
	return func(c *Config) error {
		if host == "" {
			return store.ErrorValueNotAllowed
		}
		c.host = host
		return nil
	}
}

func Port(port int) option {
	return func(c *Config) error {
		if port <= 0 {
			return store.ErrorValueNotAllowed
		}
		c.port = port
		return nil
	}
}

func Username(username string) option {
	return func(c *Config) error {
		if username == "" {
			return store.ErrorValueNotAllowed
		}
		c.username = username
		return nil
	}
}

func Password(password string) option {
	return func(c *Config) error {
		c.password = password
		return nil
	}
}

type Config struct {
	host      string
	port      int
	username  string
	password  string
	database  string
	timeout   time.Duration
	parseTime bool
}

func defaultConfig() Config {
	return Config{
		port:      DefaultPort,
		database:  dbSchema,
		timeout:   DefaultTimeout,
		parseTime: true,
	}
}

func (c Config) DSN() string {
	return fmt.Sprintf(
		dsnFmt,
		c.username,
		c.password,
		c.host,
		c.port,
		c.database,
		Utf8mb4,
		c.parseTime,
		UTC,
		c.timeout,
	)
}

func (c Config) String() string {
	return fmt.Sprintf(
		dsnFmt,
		c.username,
		"*****",
		c.host,
		c.port,
		c.database,
		Utf8mb4,
		c.parseTime,
		UTC,
		c.timeout,
	)
}
