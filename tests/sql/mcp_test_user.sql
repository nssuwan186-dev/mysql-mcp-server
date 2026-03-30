-- Dedicated MCP / integration-test user (not root).
-- Run as MySQL root after testdb exists (see init.sql).
-- Credentials: mcpuser / mcppass00

SET NAMES utf8mb4;

CREATE USER IF NOT EXISTS 'mcpuser'@'localhost' IDENTIFIED BY 'mcppass00';
CREATE USER IF NOT EXISTS 'mcpuser'@'%' IDENTIFIED BY 'mcppass00';

GRANT SELECT, SHOW VIEW, EXECUTE ON `testdb`.* TO 'mcpuser'@'localhost';
GRANT SELECT, SHOW VIEW, EXECUTE ON `testdb`.* TO 'mcpuser'@'%';

FLUSH PRIVILEGES;
