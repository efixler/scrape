package feed

import (
	"context"
	"database/sql"
	"log/slog"
	nurl "net/url"
	"time"

	"github.com/efixler/scrape/database"
)

func WithLastRequestRecord(dbh *database.DBHandle) option {
	return func(c *config) error {
		return nil
	}
}

const (
	qUpsertlastRequest = `INSERT INTO feed_refresh (url, last_request)
	VALUES(?, ?)
	ON CONFLICT(url)
	DO UPDATE SET last_request = excluded.last_request;`
)

type afterLoadFunc func(nurl.URL)
type stmtIndex int

const (
	_ stmtIndex = iota
	iUpsertLastRequest
)

func recordLastUpdatedTimeF(dbh *database.DBHandle) (afterLoadFunc, error) {
	stmt, err := dbh.Statement(
		iUpsertLastRequest,
		func(ctx context.Context, db *sql.DB) (*sql.Stmt, error) {
			return db.PrepareContext(ctx, qUpsertlastRequest)
		},
	)
	if err != nil {
		return nil, err
	}
	f := func(url nurl.URL) {
		t := time.Now().UTC().Truncate(time.Second)
		if _, err := stmt.Exec(url, t.Unix()); err != nil {
			slog.Error(
				"can't update load time for rss feed",
				"url", url,
				"time", t,
				"err", err,
			)
		}
	}
	return f, err
}
