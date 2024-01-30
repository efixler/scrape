BEGIN TRANSACTION;

CREATE DATABASE IF NOT EXISTS `scrape` DEFAULT CHARACTER SET utf8mb4 COLLATE utf8mb4_0900_ai_ci ;

DROP TABLE IF EXISTS `scrape`.`urls`;

CREATE TABLE `scrape`.`urls` (
  `id` BIGINT UNSIGNED NOT NULL,
  `url` VARCHAR(255) CHARACTER SET 'utf8mb4' COLLATE 'utf8mb4_0900_ai_ci' NOT NULL,
  `parsed_url` VARCHAR(255) CHARACTER SET 'utf8mb4' COLLATE 'utf8mb4_0900_ai_ci' NOT NULL,
  `fetch_time` INT NOT NULL,
  `expires` INT NOT NULL,
  `metadata` JSON NOT NULL,
  `content_text` TEXT CHARACTER SET 'utf8mb4' COLLATE 'utf8mb4_0900_ai_ci' NULL,
  PRIMARY KEY (`id`));

  CREATE TABLE `scrape`.`id_map` (
    `requested_id` BIGINT UNSIGNED NOT NULL,
    `canonical_id` BIGINT UNSIGNED NOT NULL,
    PRIMARY KEY (`requested_id`)
  )

  COMMIT TRANSACTION;
  