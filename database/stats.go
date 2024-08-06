package database

import (
	"database/sql"
	"log/slog"
)

type StatsProvider interface {
	Stats() (*stats, error)
}

type stats struct {
	SQL    sql.DBStats `json:"connections"`
	Driver string      `json:"driver"`
	Engine any         `json:"engine,omitempty"`
}

func (s *DBHandle) Stats() (*stats, error) {
	if s.DB == nil {
		return nil, ErrDatabaseNotOpen
	}
	stats := &stats{
		Driver: string(s.Engine.Driver()),
		SQL:    s.DB.Stats(),
	}

	if observableEngine, ok := s.Engine.(Observable); ok {
		var err error
		stats.Engine, err = observableEngine.Stats(s)
		if err != nil {
			slog.Error("error getting engine stats", "error", err)
		}
	}
	return stats, nil
}
