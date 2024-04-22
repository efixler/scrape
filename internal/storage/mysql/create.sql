-- {{.TargetSchema}}
BEGIN;
CREATE DATABASE IF NOT EXISTS `{{.TargetSchema}}` DEFAULT CHARACTER SET 'utf8mb4' COLLATE 'utf8mb4_0900_ai_ci' ;
USE {{.TargetSchema}} ;

CREATE TABLE IF NOT EXISTS `urls` (
  `id` BIGINT UNSIGNED NOT NULL,
  `url` VARCHAR(255) CHARACTER SET 'utf8mb4' COLLATE 'utf8mb4_0900_ai_ci' NOT NULL,
  `parsed_url` VARCHAR(255) CHARACTER SET 'utf8mb4' COLLATE 'utf8mb4_0900_ai_ci' NOT NULL,
  `fetch_time` BIGINT NOT NULL,
  `expires` BIGINT NOT NULL,
  `metadata` JSON NOT NULL,
  `content_text` MEDIUMTEXT CHARACTER SET 'utf8mb4' COLLATE 'utf8mb4_0900_ai_ci' NULL,
  PRIMARY KEY (`id`));

CREATE TABLE IF NOT EXISTS `id_map` (
    `requested_id` BIGINT UNSIGNED NOT NULL,
    `canonical_id` BIGINT UNSIGNED NOT NULL,
    PRIMARY KEY (`requested_id`)
);
  
-- Following two statements are added to support tracking headless
-- fetched state (or other alternate fetch methods)
-- The following cannot be executed idempotently
-- TODO: Goose migrations
ALTER TABLE urls ADD column fetch_method 
  INT UNSIGNED 
  NOT NULL DEFAULT 0;

CREATE INDEX fetch_method_expires_index ON urls (
    expires DESC,
    fetch_method ASC
);


CREATE ROLE IF NOT EXISTS scrape_app;
GRANT SELECT, INSERT, UPDATE, DELETE on {{.TargetSchema}}.* to scrape_app;
CREATE ROLE IF NOT EXISTS scrape_admin;
GRANT ALL ON {{.TargetSchema}}.* to scrape_admin;
  
COMMIT;
SET AUTOCOMMIT = 1;
