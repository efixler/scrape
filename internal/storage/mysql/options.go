package mysql

import (
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/efixler/scrape/store"
	"github.com/go-sql-driver/mysql"
)

type Charset string
type Location string
type ConnectionType string
type Collation string
type Option func(*Config) error

const (
	Utf8mb4         Charset        = "utf8mb4"
	TCP             ConnectionType = "tcp"
	Unix            ConnectionType = "unix"
	DefaultPort                    = 3306
	dbSchema                       = "scrape"
	utf8mb4General  Collation      = "utf8mb4_general_ci"
	utf8mb4Unicode9 Collation      = "utf8mb4_0900_ai_ci"
)

var (
	DefaultTimeout      = 10 * time.Second
	DefaultReadTimeout  = 30 * time.Second
	DefaultWriteTimeout = 30 * time.Second
)

func NetAddress(addr string) Option {
	return func(c *Config) error {
		if addr == "" {
			return store.ErrorValueNotAllowed
		}
		elems := strings.SplitN(addr, ":", 2)
		switch len(elems) {
		case 1:
			addr = fmt.Sprintf("%s:%d", elems[0], DefaultPort)
		case 2:
			if _, err := strconv.Atoi(elems[1]); err != nil {
				return err
			}
		}
		c.Net = string(TCP)
		c.Addr = addr
		return nil
	}
}

func Username(username string) Option {
	return func(c *Config) error {
		if username == "" {
			return store.ErrorValueNotAllowed
		}
		c.User = username
		return nil
	}
}

func Password(password string) Option {
	return func(c *Config) error {
		c.Passwd = password
		return nil
	}
}

func Schema(name string) Option {
	return func(c *Config) error {
		c.DBName = name
		return nil
	}
}

func WithoutSchema() Option {
	return func(c *Config) error {
		c.DBName = ""
		return nil
	}
}

type Config struct {
	mysql.Config
}

func defaultConfig() Config {
	cfg := mysql.NewConfig()
	cfg.Net = string(TCP)
	cfg.DBName = dbSchema
	cfg.Loc = time.UTC
	cfg.Collation = string(utf8mb4Unicode9)
	cfg.Timeout = DefaultTimeout
	cfg.ReadTimeout = DefaultReadTimeout
	cfg.WriteTimeout = DefaultWriteTimeout
	cfg.ParseTime = true
	cfg.MultiStatements = true
	return Config{*cfg}
}

func (c Config) DSN() string {
	return c.Config.FormatDSN()
}

func (c Config) String() string {
	cp := c.Config
	cp.Passwd = "*****"
	return cp.FormatDSN()
}
