package sqlite

import (
	"log/slog"
	"path/filepath"
	"syscall"
	"time"

	"github.com/efixler/scrape/database"
)

const (
	minStatsInterval = 1 * time.Minute
)

type Stats struct {
	SqliteVersion string           `json:"sqlite_version"`
	PageCount     int              `json:"page_count"`
	PageSize      int              `json:"page_size"`
	UnusedPages   int              `json:"unused_pages"`
	MaxPageCount  int              `json:"max_page_count"`
	Filesystem    *FilesystemStats `json:"fs,omitempty"`
	DBStats       any              `json:"db_stats,omitempty"`
	fetchTime     time.Time
}

func (s *Stats) DatabaseSizeMB() int {
	return int(float64(s.PageCount*s.PageSize) / (1024 * 1024))
}

func (s Stats) expired() bool {
	return time.Since(s.fetchTime) > minStatsInterval
}

// Implements the store.Observable interface. Return value intended to be
// included in JSON outputs. For introspection of the results, type assert
// to *sqlite.Stats.
func (s *SQLite) Stats(dbh *database.DBHandle) (any, error) {
	if s.stats != nil && !s.stats.expired() {
		return s.stats, nil
	}

	var pageCount, pageSize, unusedPages, maxPageCount int

	err := dbh.DB.QueryRowContext(dbh.Ctx, "PRAGMA page_count;").Scan(&pageCount)
	if err != nil {
		return nil, err
	}
	err = dbh.DB.QueryRowContext(dbh.Ctx, "PRAGMA page_size;").Scan(&pageSize)
	if err != nil {
		return nil, err
	}
	err = dbh.DB.QueryRowContext(dbh.Ctx, "PRAGMA freelist_count;").Scan(&unusedPages)
	if err != nil {
		return nil, err
	}
	err = dbh.DB.QueryRowContext(dbh.Ctx, "PRAGMA max_page_count;").Scan(&maxPageCount)
	if err != nil {
		return nil, err
	}

	if s.stats == nil {
		var sqliteVersion string
		err := dbh.DB.QueryRowContext(dbh.Ctx, "SELECT sqlite_version();").Scan(&sqliteVersion)
		if err != nil {
			return nil, err
		}
		s.stats = &Stats{SqliteVersion: sqliteVersion}
	}
	s.stats.PageCount = pageCount
	s.stats.PageSize = pageSize
	s.stats.UnusedPages = unusedPages
	s.stats.MaxPageCount = maxPageCount
	s.stats.Filesystem = s.filesystemStats()
	// s.stats.DBStats, _ = s.WebPages.Stats()
	s.stats.fetchTime = time.Now()
	return s.stats, nil
}

type FilesystemStats struct {
	Path    string `json:"path"`
	TotalMB uint   `json:"total_mb"`
	UsedMB  uint   `json:"used_mb"`
	FreeMB  uint   `json:"free_mb"`
	AvailMB uint   `json:"avail_mb"`
}

func (s SQLite) filesystemStats() *FilesystemStats {
	if s.config.filename == InMemoryDBName {
		return nil
	}
	dir := filepath.Dir(s.config.filename)
	var stat syscall.Statfs_t
	if err := syscall.Statfs(dir, &stat); err != nil {
		slog.Warn("Error getting filesystem stats", "error", err)
		return nil
	}
	return &FilesystemStats{
		Path:    dir,
		TotalMB: uint(stat.Blocks * uint64(stat.Bsize) / (1024 * 1024)),
		FreeMB:  uint(stat.Bfree * uint64(stat.Bsize) / (1024 * 1024)),
		UsedMB:  uint((stat.Blocks - stat.Bfree) * uint64(stat.Bsize) / (1024 * 1024)),
		AvailMB: uint(stat.Bavail * uint64(stat.Bsize) / (1024 * 1024)),
	}
}
