-- {{.TargetSchema}}
BEGIN;
CREATE DATABASE IF NOT EXISTS `{{.TargetSchema}}` DEFAULT CHARACTER SET 'utf8mb4' COLLATE 'utf8mb4_0900_ai_ci' ;
CREATE ROLE IF NOT EXISTS scrape_app;
GRANT SELECT, INSERT, UPDATE, DELETE on {{.TargetSchema}}.* to scrape_app;
CREATE ROLE IF NOT EXISTS scrape_admin;
GRANT ALL ON {{.TargetSchema}}.* to scrape_admin;

USE {{.TargetSchema}} ;

COMMIT;
SET AUTOCOMMIT = 1;
