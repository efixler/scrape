package mysql

import (
	"fmt"
	"time"
)

type Charset string
type Location string

const (
	Utf8mb4  Charset  = "utf8mb4"
	UTC      Location = "UTC"
	dsnFmt            = "%s:%s@tcp(%s:%d)/%s?charset=%s&parseTime=%t&loc=%s&timeout=%s"
	dbSchema          = "scrape"
)

var (
	DefaultTimeout = 10 * time.Second
)

type Options struct {
	Host      string
	Port      int
	Username  string
	Password  string
	database  string
	timeout   time.Duration
	parseTime bool
}

func DefaultOptions() Options {
	return Options{
		Port:      3306,
		database:  dbSchema,
		timeout:   DefaultTimeout,
		parseTime: true,
	}
}

func (o Options) DSN() string {
	return fmt.Sprintf(
		dsnFmt,
		o.Username,
		o.Password,
		o.Host,
		o.Port,
		o.database,
		Utf8mb4,
		o.parseTime,
		UTC,
		o.timeout,
	)
}

func (o Options) String() string {
	return fmt.Sprintf(
		dsnFmt,
		o.Username,
		"*****",
		o.Host,
		o.Port,
		o.database,
		Utf8mb4,
		o.parseTime,
		UTC,
		o.timeout,
	)
}
