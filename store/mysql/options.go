package mysql

import (
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/efixler/scrape/store"
)

type Charset string
type Location string
type ConnectionType string
type option func(*Config) error

const (
	Utf8mb4     Charset        = "utf8mb4"
	UTC         Location       = "UTC"
	TCP         ConnectionType = "tcp"
	Unix        ConnectionType = "unix"
	DefaultPort                = 3306
	dsnFmt                     = "%s:%s@%s(%s:%d)/%s?charset=%s&parseTime=%t&loc=%s&timeout=%s&multiStatements=%t"
	dbSchema                   = "scrape"
)

var (
	DefaultTimeout      = 10 * time.Second
	DefaultReadTimeout  = 30 * time.Second
	DefaultWriteTimeout = 30 * time.Second
)

func Address(addr string) option {
	return func(c *Config) error {
		elems := strings.SplitN(addr, ":", 2)
		err := Host(elems[0])(c)
		if err != nil {
			return err
		}
		var port int
		switch len(elems) {
		case 1:
			port = DefaultPort
		case 2:
			port, err = strconv.Atoi(elems[1])
			if err != nil {
				return err
			}
		}
		err = Port(port)(c)
		if err != nil {
			return err
		}
		return nil
	}
}

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
	connectionType     ConnectionType
	host               string
	port               int
	username           string
	password           string
	database           string
	timeout            time.Duration
	readTimeout        time.Duration
	writeTimeout       time.Duration
	parseTime          bool
	multipleStatements bool
}

func defaultConfig() Config {
	return Config{
		connectionType:     TCP,
		port:               DefaultPort,
		database:           dbSchema,
		timeout:            DefaultTimeout,
		readTimeout:        DefaultReadTimeout,
		writeTimeout:       DefaultWriteTimeout,
		parseTime:          true,
		multipleStatements: true,
	}
}

func (c Config) DSN() string {
	return fmt.Sprintf(
		dsnFmt,
		c.username,
		c.password,
		c.connectionType,
		c.host,
		c.port,
		c.database,
		Utf8mb4,
		c.parseTime,
		UTC,
		c.timeout,
		c.multipleStatements,
	)
}

func (c Config) String() string {
	return fmt.Sprintf(
		dsnFmt,
		c.username,
		"*****",
		c.connectionType,
		c.host,
		c.port,
		c.database,
		Utf8mb4,
		c.parseTime,
		UTC,
		c.timeout,
		c.multipleStatements,
	)
}
