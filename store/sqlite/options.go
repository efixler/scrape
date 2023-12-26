package sqlite

import (
	"fmt"
	"time"
)

const (
	DEFAULT_BUSY_TIMEOUT = 5 * time.Second
	DEFAULT_JOURNAL_MODE = "WAL"
	DEFAULT_CACHE_SIZE   = 20000
	smallCacheSize       = 2000 // This is actually the sqlite default
	SQLITE_SYNC_OFF      = "OFF"
	SQLITE_SYNC_NORMAL   = "NORMAL"
	DEFAULT_SYNC         = SQLITE_SYNC_OFF
)

type sqliteOptions struct {
	busyTimeout time.Duration
	journalMode string
	cacheSize   int
	synchronous string
}

func (o sqliteOptions) String() string {
	return fmt.Sprintf(
		"_busy_timeout=%d&_journal_mode=%s&_cache_size=%d&_sync=%s",
		o.busyTimeout,
		o.journalMode,
		o.cacheSize,
		o.synchronous,
	)
}

// Returns an options set tuned for on-disk databases
func DefaultOptions() sqliteOptions {
	return sqliteOptions{
		busyTimeout: DEFAULT_BUSY_TIMEOUT,
		journalMode: DEFAULT_JOURNAL_MODE,
		cacheSize:   DEFAULT_CACHE_SIZE,
		synchronous: DEFAULT_SYNC,
	}
}

// Returns an options set tuned for in-memory databases
func InMemoryOptions() sqliteOptions {
	return sqliteOptions{
		busyTimeout: DEFAULT_BUSY_TIMEOUT,
		journalMode: DEFAULT_JOURNAL_MODE,
		cacheSize:   smallCacheSize,
		synchronous: SQLITE_SYNC_NORMAL,
	}
}
