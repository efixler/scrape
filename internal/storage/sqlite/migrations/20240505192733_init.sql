-- +goose Up
-- +goose StatementBegin
PRAGMA foreign_keys = off;
PRAGMA page_size = 16384; -- default is 4096 but text records are big
PRAGMA temp_store = MEMORY;
PRAGMA auto_vacuum = INCREMENTAL;

CREATE TABLE IF NOT EXISTS id_map (
    requested_id INTEGER PRIMARY KEY ON CONFLICT REPLACE
                         NOT NULL,
    canonical_id INTEGER NOT NULL
)
WITHOUT ROWID,
STRICT;

CREATE TABLE IF NOT EXISTS urls (
    id           INTEGER PRIMARY KEY ON CONFLICT REPLACE
                         NOT NULL,
    url          TEXT    NOT NULL
                         COLLATE NOCASE,
    parsed_url   TEXT    NOT NULL,
    fetch_time   INTEGER DEFAULT (unixepoch() ),
    fetch_method INTEGER NOT NULL DEFAULT 0,
    expires      INTEGER DEFAULT (unixepoch() + 86400),
    metadata     TEXT,
    content_text TEXT
)
WITHOUT ROWID,
STRICT;

CREATE INDEX IF NOT EXISTS fetch_method_expires_index ON urls (
    expires DESC,
    fetch_method ASC
);

-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TABLE IF EXISTS urls;
DROP TABLE IF EXISTS id_map;
-- +goose StatementEnd
