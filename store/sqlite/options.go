package sqlite

import (
	"fmt"
	"time"
)

type JournalMode string
type SyncMode string
type AccessMode string

const (
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

type sqliteOptions struct {
	filename    string
	busyTimeout time.Duration
	journalMode JournalMode
	cacheSize   int
	synchronous SyncMode
	accessMode  AccessMode
}

func (o sqliteOptions) DSN() string {
	return o.String()
}

func (o sqliteOptions) String() string {
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

// Returns an options set tuned for on-disk databases
func DefaultOptions() sqliteOptions {
	return sqliteOptions{
		busyTimeout: FiveSecondDuration,
		journalMode: JournalModeWAL,
		cacheSize:   BigCacheSize,
		synchronous: SyncOff,
		accessMode:  AccessModeRWC,
	}
}

// Returns an options set tuned for in-memory databases
func InMemoryOptions() sqliteOptions {
	return sqliteOptions{
		filename:    InMemoryDBName, // this is _always_ the name for in-memory DBs
		busyTimeout: FiveSecondDuration,
		journalMode: JournalModeOff,
		cacheSize:   NormalCacheSize,
		synchronous: SyncNormal,
		accessMode:  AccessModeMemory,
	}
}
