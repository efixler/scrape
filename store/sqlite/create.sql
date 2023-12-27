--
-- File generated with SQLiteStudio v3.4.4 on Thu Dec 14 22:50:06 2023
--
-- Text encoding used: UTF-8
--
PRAGMA foreign_keys = off;
BEGIN TRANSACTION;

-- Table: id_map

DROP TABLE IF EXISTS id_map;

CREATE TABLE id_map (
    requested_id INTEGER PRIMARY KEY ON CONFLICT REPLACE
                         NOT NULL,
    canonical_id INTEGER NOT NULL
)
WITHOUT ROWID,
STRICT;


-- Table: urls
DROP TABLE IF EXISTS urls;

CREATE TABLE urls (
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

COMMIT TRANSACTION;
PRAGMA wal_checkpoint(RESTART);
