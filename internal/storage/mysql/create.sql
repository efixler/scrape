-- {{.DBName}}
BEGIN;
CREATE DATABASE IF NOT EXISTS `{{.DBName}}` DEFAULT CHARACTER SET 'utf8mb4' COLLATE 'utf8mb4_0900_ai_ci' ;
USE {{.DBName}} ;

DROP TABLE IF EXISTS `urls`;

CREATE TABLE `urls` (
  `id` BIGINT UNSIGNED NOT NULL,
  `url` VARCHAR(255) CHARACTER SET 'utf8mb4' COLLATE 'utf8mb4_0900_ai_ci' NOT NULL,
  `parsed_url` VARCHAR(255) CHARACTER SET 'utf8mb4' COLLATE 'utf8mb4_0900_ai_ci' NOT NULL,
  `fetch_time` INT NOT NULL,
  `expires` INT NOT NULL,
  `metadata` JSON NOT NULL,
  `content_text` TEXT CHARACTER SET 'utf8mb4' COLLATE 'utf8mb4_0900_ai_ci' NULL,
  PRIMARY KEY (`id`));

DROP TABLE IF EXISTS `id_map`;

  CREATE TABLE `id_map` (
    `requested_id` BIGINT UNSIGNED NOT NULL,
    `canonical_id` BIGINT UNSIGNED NOT NULL,
    PRIMARY KEY (`requested_id`)
  );
  
  CREATE ROLE IF NOT EXISTS scrape_app;
  GRANT SELECT, UPDATE, DELETE on {{.DBName}}.* to scrape_app;
  CREATE ROLE IF NOT EXISTS scrape_admin;
  GRANT ALL ON {{.DBName}}.* to scrape_admin;
  
COMMIT;
SET AUTOCOMMIT = 1;
