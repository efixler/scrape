-- This migration adds per-domain settings to the database.
-- +goose Up
-- +goose StatementBegin

CREATE TABLE IF NOT EXISTS `domain_settings` (
    `domain` VARCHAR(255) PRIMARY KEY NOT NULL,
    `sitename` VARCHAR(255),
    `fetch_client` VARCHAR(255),
    `user_agent` VARCHAR(512),
    `headers` JSON
);

-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TABLE IF EXISTS domain_settings;
-- +goose StatementEnd
