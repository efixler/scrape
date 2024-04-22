--
--
-- Text encoding used: UTF-8
--
PRAGMA foreign_keys = off;
PRAGMA page_size = 16384; -- default is 4096 but text records are big
PRAGMA temp_store = MEMORY;
PRAGMA auto_vacuum = INCREMENTAL;
BEGIN TRANSACTION;

-- Table: id_map

CREATE TABLE IF NOT EXISTS id_map (
    requested_id INTEGER PRIMARY KEY ON CONFLICT REPLACE
                         NOT NULL,
    canonical_id INTEGER NOT NULL
)
WITHOUT ROWID,
STRICT;


-- Table: urls
CREATE TABLE IF NOT EXISTS urls (
    id           INTEGER PRIMARY KEY ON CONFLICT REPLACE
                         NOT NULL,
    url          TEXT    NOT NULL
                         COLLATE NOCASE,
    parsed_url   TEXT    NOT NULL,
    fetch_time   INTEGER DEFAULT (unixepoch() ),
    expires      INTEGER DEFAULT (unixepoch() + 86400),
    metadata     TEXT,
    content_text TEXT
)
WITHOUT ROWID,
STRICT;


-- Following two statements are added to support tracking headless
-- fetched state (or other alternate fetch methods)
-- The following cannot be executed idempotently
-- TODO: Goose migrations
ALTER TABLE urls ADD column fetch_method INTEGER NOT NULL DEFAULT 0;

CREATE INDEX IF NOT EXISTS fetch_method_expires_index ON urls (
    expires DESC,
    fetch_method ASC
);

COMMIT TRANSACTION;
PRAGMA wal_checkpoint(RESTART);

