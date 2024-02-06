package sqlite

import (
	"errors"
	"fmt"
	"os"
	"time"
)

type JournalMode string
type SyncMode string
type AccessMode string
type option func(*config) error

const (
	EnvDBPath                      = "SCRAPE_DB"
	FiveSecondDuration             = 5 * time.Second
	JournalModeWAL     JournalMode = "WAL"
	JournalModeMemory  JournalMode = "MEMORY"
	JournalModeOff     JournalMode = "OFF"
	BigCacheSize                   = 20000
	NormalCacheSize                = 2000 // This is actually the sqlite default
	SyncOff            SyncMode    = "OFF"
	SyncNormal         SyncMode    = "NORMAL"
	AccessModeRWC      AccessMode  = "rwc"
	AccessModeMemory   AccessMode  = "memory"
)

func InMemoryDB() option {
	return func(c *config) error {
		c.filename = InMemoryDBName
		c.accessMode = AccessModeMemory
		c.busyTimeout = FiveSecondDuration
		c.journalMode = JournalModeOff
		c.cacheSize = NormalCacheSize
		c.synchronous = SyncNormal
		return nil
	}
}

func Defaults() option {
	return func(c *config) error {
		c.filename = DefaultDatabase
		c.accessMode = AccessModeRWC
		c.busyTimeout = FiveSecondDuration
		c.journalMode = JournalModeWAL
		c.cacheSize = BigCacheSize
		c.synchronous = SyncOff
		return nil
	}
}

func WithFileOrEnv(filename string) option {
	return func(c *config) error {
		if filename == "" {
			filename = os.Getenv(EnvDBPath)
		}
		return WithFile(filename)(c)
	}
}

func WithFile(filename string) option {
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
				return errors.Join(ErrCantCreateDatabase, err)
			}
			c.filename = resolvedPath
		}
		return nil
	}
}

type config struct {
	filename    string
	busyTimeout time.Duration
	journalMode JournalMode
	cacheSize   int
	synchronous SyncMode
	accessMode  AccessMode
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

func (o config) IsInMemory() bool {
	return o.filename == InMemoryDBName
}
