package sqlite

import "time"

const (
	minStatsInterval = 1 * time.Minute
)

type Stats struct {
	SqliteVersion string
	PageCount     int `json:"page_count"`
	PageSize      int `json:"page_size"`
	UnusedPages   int `json:"unused_pages"`
	MaxPageCount  int `json:"max_page_count"`
	fetchTime     time.Time
}

func (s *Stats) DatabaseSizeMB() int {
	return int(float64(s.PageCount*s.PageSize) / (1024 * 1024))
}

func (s Stats) expired() bool {
	return time.Since(s.fetchTime) > minStatsInterval
}

func (s *SqliteStore) Stats() (*Stats, error) {
	if s.stats != nil && !s.stats.expired() {
		return s.stats, nil
	}

	var pageCount, pageSize, unusedPages, maxPageCount int

	err := s.DB.QueryRowContext(s.Ctx, "PRAGMA page_count;").Scan(&pageCount)
	if err != nil {
		return nil, err
	}
	err = s.DB.QueryRowContext(s.Ctx, "PRAGMA page_size;").Scan(&pageSize)
	if err != nil {
		return nil, err
	}
	err = s.DB.QueryRowContext(s.Ctx, "PRAGMA freelist_count;").Scan(&unusedPages)
	if err != nil {
		return nil, err
	}
	err = s.DB.QueryRowContext(s.Ctx, "PRAGMA max_page_count;").Scan(&maxPageCount)
	if err != nil {
		return nil, err
	}

	if s.stats == nil {
		var sqliteVersion string
		err := s.DB.QueryRowContext(s.Ctx, "SELECT sqlite_version();").Scan(&sqliteVersion)
		if err != nil {
			return nil, err
		}
		s.stats = &Stats{SqliteVersion: sqliteVersion}
	}
	s.stats.PageCount = pageCount
	s.stats.PageSize = pageSize
	s.stats.UnusedPages = unusedPages
	s.stats.MaxPageCount = maxPageCount
	s.stats.fetchTime = time.Now()
	return s.stats, nil
}
