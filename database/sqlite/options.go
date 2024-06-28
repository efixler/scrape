package sqlite

import (
	"errors"
	"fmt"
	"os"
	"time"

	"github.com/efixler/scrape/store"
)

type JournalMode string
type SyncMode string
type AccessMode string
type Option func(*config) error

const (
	EnvDBPath                       = "SCRAPE_DB"
	DefaultBusyTimeout              = 5 * time.Second
	DefaultQueryTimeout             = 30 * time.Second
	JournalModeWAL      JournalMode = "WAL"
	JournalModeMemory   JournalMode = "MEMORY"
	JournalModeOff      JournalMode = "OFF"
	BigCacheSize                    = 20000
	NormalCacheSize                 = 2000 // This is actually the sqlite default
	SyncOff             SyncMode    = "OFF"
	SyncNormal          SyncMode    = "NORMAL"
	AccessModeRWC       AccessMode  = "rwc"
	AccessModeMemory    AccessMode  = "memory"
)

func InMemoryDB() Option {
	return func(c *config) error {
		c.filename = InMemoryDBName
		c.accessMode = AccessModeMemory
		c.busyTimeout = DefaultBusyTimeout
		c.journalMode = JournalModeOff
		c.cacheSize = NormalCacheSize
		c.synchronous = SyncNormal
		c.maxConnections = 1 //in-memory DBs support 1 connection only
		c.connMaxLifetime = -1
		return nil
	}
}

// Defaults always get applied in the New() function
func Defaults() Option {
	return func(c *config) error {
		c.filename = DefaultDatabase
		c.accessMode = AccessModeRWC
		c.busyTimeout = DefaultBusyTimeout
		c.journalMode = JournalModeWAL
		c.cacheSize = BigCacheSize
		c.synchronous = SyncOff
		c.queryTimeout = DefaultQueryTimeout
		return nil
	}
}

func WithFileOrEnv(filename string) Option {
	return func(c *config) error {
		if filename == "" {
			filename = os.Getenv(EnvDBPath)
		}
		return File(filename)(c)
	}
}

func File(filename string) Option {
	return func(c *config) error {
		if resolvedPath, err := dbPath(filename); err != nil {
			switch err {
			case ErrIsInMemory:
				return InMemoryDB()(c)
			default:
				// if there was an error here we won't be able to open or create
				return err
			}
		} else {
			err = assertPathTo(resolvedPath)
			if err != nil {
				return errors.Join(store.ErrCantCreateDatabase, err)
			}
			c.filename = resolvedPath
		}
		return nil
	}
}

func WithQueryTimeout(timeout time.Duration) Option {
	return func(c *config) error {
		c.queryTimeout = timeout
		return nil
	}
}

func WithoutAutoCreate() Option {
	return func(c *config) error {
		c.noAutoCreate = true
		return nil
	}
}

type config struct {
	filename        string
	busyTimeout     time.Duration // SQLite's busy timeout specifies the time to wait if a table is locked
	queryTimeout    time.Duration
	journalMode     JournalMode
	cacheSize       int
	synchronous     SyncMode
	accessMode      AccessMode
	maxConnections  int           // 0 = use driver defaults
	connMaxLifetime time.Duration // 0 = use driver defaults
	noAutoCreate    bool
}

func (o config) DSN() string {
	return o.String()
}

func (o config) String() string {
	return fmt.Sprintf(
		"file:%s?mode=%s&_busy_timeout=%d&_journal_mode=%s&_cache_size=%d&_sync=%s",
		o.filename,
		o.accessMode,
		o.busyTimeout.Milliseconds(),
		o.journalMode,
		o.cacheSize,
		o.synchronous,
	)
}

func (o config) QueryTimeout() time.Duration {
	return o.queryTimeout
}

func (o config) MaxConnections() int {
	return o.maxConnections
}

func (o config) ConnMaxLifetime() time.Duration {
	return o.connMaxLifetime
}

func (o config) IsInMemory() bool {
	return o.filename == InMemoryDBName
}

func (o config) databaseExists() bool {
	if o.IsInMemory() || !exists(o.filename) {
		return false
	}
	return true
}

func (o config) autoCreate() bool {
	return !o.noAutoCreate
}
