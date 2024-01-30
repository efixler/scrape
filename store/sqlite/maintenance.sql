PRAGMA foreign_keys = OFF;
DELETE from urls where expires < strftime('%s', 'now');
PRAGMA page_size = 32768;
PRAGMA journal_mode = WAL;
PRAGMA wal_checkpoint(TRUNCATE);
PRAGMA auto_vacuum = INCREMENTAL;
PRAGMA vacuum;
PRAGMA optimize;