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
    parsed_url_id INTEGER PRIMARY KEY,
    url_id        INTEGER
)
WITHOUT ROWID,
STRICT;


-- Table: urls
DROP TABLE IF EXISTS urls;

CREATE TABLE urls (
    id           INTEGER PRIMARY KEY ON CONFLICT ABORT
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


-- Index: url_id_index
DROP INDEX IF EXISTS url_id_index;

CREATE INDEX url_id_index ON id_map (
    url_id
);


COMMIT TRANSACTION;
PRAGMA foreign_keys = on;
