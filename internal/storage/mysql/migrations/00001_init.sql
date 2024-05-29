-- +goose Up
-- +goose StatementBegin

CREATE TABLE IF NOT EXISTS `urls` (
  `id` BIGINT UNSIGNED NOT NULL,
  `url` VARCHAR(255) CHARACTER SET 'utf8mb4' COLLATE 'utf8mb4_0900_ai_ci' NOT NULL,
  `parsed_url` VARCHAR(255) CHARACTER SET 'utf8mb4' COLLATE 'utf8mb4_0900_ai_ci' NOT NULL,
  `fetch_time` BIGINT NOT NULL,
  `fetch_method` INT UNSIGNED NOT NULL DEFAULT 0,
  `expires` BIGINT NOT NULL,
  `metadata` JSON NOT NULL,
  `content_text` MEDIUMTEXT CHARACTER SET 'utf8mb4' COLLATE 'utf8mb4_0900_ai_ci' NULL,
  PRIMARY KEY (`id`));

CREATE TABLE IF NOT EXISTS `id_map` (
    `requested_id` BIGINT UNSIGNED NOT NULL,
    `canonical_id` BIGINT UNSIGNED NOT NULL,
    PRIMARY KEY (`requested_id`)
);

CREATE INDEX fetch_method_expires_index ON urls (
    expires DESC,
    fetch_method ASC
);

-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
-- USE `scrape_test`;
DROP TABLE IF EXISTS `urls`;
DROP TABLE IF EXISTS `id_map`;
-- +goose StatementEnd
