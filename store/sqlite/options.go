package sqlite

import (
	"fmt"
	"time"
)

type WALMode string
type SyncMode string

const (
	FiveSecondDuration          = 5 * time.Second
	WALJournalMode     WALMode  = "WAL"
	BigCacheSize                = 20000
	NormalCacheSize             = 2000 // This is actually the sqlite default
	SyncOff            SyncMode = "OFF"
	SyncNormal         SyncMode = "NORMAL"
)

type sqliteOptions struct {
	busyTimeout time.Duration
	journalMode WALMode
	cacheSize   int
	synchronous SyncMode
}

func (o sqliteOptions) String() string {
	return fmt.Sprintf(
		"_busy_timeout=%d&_journal_mode=%s&_cache_size=%d&_sync=%s",
		o.busyTimeout.Milliseconds(),
		o.journalMode,
		o.cacheSize,
		o.synchronous,
	)
}

// Returns an options set tuned for on-disk databases
func DefaultOptions() sqliteOptions {
	return sqliteOptions{
		busyTimeout: FiveSecondDuration,
		journalMode: WALJournalMode,
		cacheSize:   BigCacheSize,
		synchronous: SyncOff,
	}
}

// Returns an options set tuned for in-memory databases
func InMemoryOptions() sqliteOptions {
	return sqliteOptions{
		busyTimeout: FiveSecondDuration,
		journalMode: WALJournalMode,
		cacheSize:   NormalCacheSize,
		synchronous: SyncNormal,
	}
}
