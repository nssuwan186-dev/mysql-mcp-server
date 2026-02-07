Sakila Test Steps (multi-version matrix)

Prereqs
- Docker engine running
- Repo root: /Users/askdba/Documents/GitHub/mysql-mcp-server

Steps
1) Start MySQL 8.4 from compose
   docker compose -f docker-compose.test.yml up -d mysql84

2) Wait for mysql84 to be healthy
   docker inspect -f "{{.State.Health.Status}}" mysql-mcp-test-84

3) Run Sakila tests on MySQL 8.4
   MYSQL_SAKILA_DSN="root:testpass@tcp(localhost:3307)/sakila?parseTime=true" \
     go test -tags=integration ./tests/integration -run Sakila -v

4) Start MySQL 9.0 from compose
   docker compose -f docker-compose.test.yml up -d mysql90

5) Ensure mysql90 is healthy
   docker inspect -f "{{.State.Health.Status}}" mysql-mcp-test-90

6) Start MariaDB 11.4 from compose
   docker compose -f docker-compose.test.yml up -d mariadb11

7) Ensure mariadb11 is healthy
   docker inspect -f "{{.State.Health.Status}}" mysql-mcp-test-mariadb-11

8) Run MySQL 8.0 on alternate port (if 3306 is in use)
   docker run -d --name mysql-mcp-test-80-alt \
     -e MYSQL_ROOT_PASSWORD=testpass \
     -e MYSQL_DATABASE=testdb \
     -e MYSQL_USER=testuser \
     -e MYSQL_PASSWORD=testpass \
     -p 3311:3306 \
     -v mysql80_alt_data:/var/lib/mysql \
     -v "$(pwd)/tests/sql/init.sql":/docker-entrypoint-initdb.d/01-init.sql:ro \
     -v "$(pwd)/tests/sql/sakila-schema.sql":/docker-entrypoint-initdb.d/02-sakila-schema.sql:ro \
     -v "$(pwd)/tests/sql/sakila-data.sql":/docker-entrypoint-initdb.d/03-sakila-data.sql:ro \
     mysql:8.0 \
     --default-authentication-plugin=mysql_native_password \
     --character-set-server=utf8mb4 \
     --collation-server=utf8mb4_unicode_ci

9) Wait for mysql80-alt to be ready
   docker exec mysql-mcp-test-80-alt \
     mysqladmin ping -h localhost -u root -ptestpass

10) Run Sakila tests on MySQL 8.0 (alt port)
    MYSQL_SAKILA_DSN="root:testpass@tcp(localhost:3311)/sakila?parseTime=true" \
      go test -tags=integration ./tests/integration -run Sakila -v

11) Run Sakila tests on MySQL 9.0
    MYSQL_SAKILA_DSN="root:testpass@tcp(localhost:3308)/sakila?parseTime=true" \
      go test -tags=integration ./tests/integration -run Sakila -v

12) Run Sakila tests on MariaDB 11.4
    MYSQL_SAKILA_DSN="root:testpass@tcp(localhost:3310)/sakila?parseTime=true" \
      go test -tags=integration ./tests/integration -run Sakila -v

Cleanup
13) Stop and remove compose containers, network, and volumes
    docker compose -f docker-compose.test.yml down -v

14) Remove the MySQL 8.0 alt container and volume
    docker rm -f mysql-mcp-test-80-alt
    docker volume rm mysql80_alt_data
