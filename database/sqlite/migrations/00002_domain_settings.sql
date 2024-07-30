-- This migration adds per-domain settings to the database.
-- +goose Up
-- +goose StatementBegin
CREATE TABLE IF NOT EXISTS domain_settings (
    domain TEXT PRIMARY KEY ON CONFLICT REPLACE NOT NULL,
    sitename TEXT,
    fetch_client TEXT,
    user_agent TEXT,
    headers  TEXT,
    check (length(domain) <= 255)
    check (length(sitename) <= 255)
    check (length(fetch_client) <= 255)
    check (length(user_agent) <= 1024)
    check (json_valid(headers))
) WITHOUT ROWID;
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TABLE IF EXISTS domain_settings;
-- +goose StatementEnd
