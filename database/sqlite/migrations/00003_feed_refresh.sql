-- This migration adds a feed_refresh table to the database.
-- +goose Up
-- +goose StatementBegin

CREATE TABLE IF NOT EXISTS feed_refresh (
    url TEXT PRIMARY KEY NOT NULL ON CONFLICT REPLACE COLLATE NOCASE,
    last_request INTEGER NOT NULL DEFAULT (unixepoch() ),
    refresh_interval INTEGER NOT NULL DEFAULT (3600 * 12),
    last_refresh INTEGER NOT NULL DEFAULT 0,
    idle_timeout INTEGER NOT NULL DEFAULT (86400 * 7)
)
STRICT;

CREATE INDEX IF NOT EXISTS feed_refresh_url_index ON feed_refresh (
    url ASC
);

CREATE INDEX IF NOT EXISTS feed_refresh_time_index ON feed_refresh (
    last_refresh ASC,
    refresh_interval ASC,
    url ASC
);

-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TABLE IF EXISTS feed_refresh;
-- +goose StatementEnd