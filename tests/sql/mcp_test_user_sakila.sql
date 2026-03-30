-- Extra grants for Sakila (run after sakila schema/data are loaded).
-- Requires mcp_test_user.sql to have run first.

SET NAMES utf8mb4;

GRANT SELECT, SHOW VIEW, EXECUTE ON `sakila`.* TO 'mcpuser'@'localhost';
GRANT SELECT, SHOW VIEW, EXECUTE ON `sakila`.* TO 'mcpuser'@'%';

FLUSH PRIVILEGES;
