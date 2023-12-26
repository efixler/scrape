package sqlite

import (
	"fmt"
	"time"
)

type SqliteOptions struct {
	busyTimeout time.Duration
	journalMode string
	cacheSize   int
	synchronous string
}

func (o SqliteOptions) String() string {
	return fmt.Sprintf(
		"_busy_timeout=%d&_journal_mode=%s&_cache_size=%d&_sync=%s",
		o.busyTimeout,
		o.journalMode,
		o.cacheSize,
		o.synchronous,
	)
}

func DefaultOptions() SqliteOptions {
	return SqliteOptions{
		busyTimeout: DEFAULT_BUSY_TIMEOUT,
		journalMode: DEFAULT_JOURNAL_MODE,
		cacheSize:   DEFAULT_CACHE_SIZE,
		synchronous: DEFAULT_SYNC,
	}
}

func InMemoryOptions() SqliteOptions {
	return SqliteOptions{
		busyTimeout: DEFAULT_BUSY_TIMEOUT,
		journalMode: DEFAULT_JOURNAL_MODE,
		cacheSize:   SMALL_CACHE_SIZE,
		synchronous: SQLITE_SYNC_NORMAL,
	}
}
