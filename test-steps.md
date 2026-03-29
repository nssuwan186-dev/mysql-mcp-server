1. Test steps run (Sakila integration matrix)

Prereqs
- Docker engine running
- Bash (helpers use POSIX `while`, `sleep`, and arithmetic)
- Repository root: `<repo-root>` — replace with your clone path, or run `cd` to the directory that contains `docker-compose.test.yml` (same as `$(pwd)` after `cd` there)

### Bash helpers (run once per shell session; requires `cd` to `<repo-root>` first)

```bash
wait_docker_healthy() {
  local cname=$1
  local max_attempts=${2:-60}
  local sleep_s=${3:-2}
  local i=0
  while [ "$i" -lt "$max_attempts" ]; do
    st=$(docker inspect -f '{{.State.Health.Status}}' "$cname" 2>/dev/null || echo "unknown")
    echo "  $cname health: $st (attempt $((i+1))/$max_attempts)"
    [ "$st" = "healthy" ] && return 0
    i=$((i+1))
    sleep "$sleep_s"
  done
  echo "error: $cname did not become healthy within $((max_attempts * sleep_s))s" >&2
  return 1
}

wait_mysqladmin_ping() {
  local name=$1
  local max_attempts=${2:-60}
  local sleep_s=${3:-3}
  local i=0
  while [ "$i" -lt "$max_attempts" ]; do
    echo "  $name: mysqladmin ping (attempt $((i+1))/$max_attempts)"
    if docker exec "$name" mysqladmin ping -h localhost -u root -ptestpass 2>/dev/null; then
      echo "  $name: MySQL is ready."
      return 0
    fi
    i=$((i+1))
    sleep "$sleep_s"
  done
  echo "error: $name did not respond to mysqladmin ping within $((max_attempts * sleep_s))s" >&2
  return 1
}
```

Steps (from `<repo-root>`)

1) Start MySQL 8.4 from compose
   docker compose -f docker-compose.test.yml up -d mysql84

2) Wait for mysql84 to be healthy (`mysql-mcp-test-84`)
   wait_docker_healthy mysql-mcp-test-84

3) Run Sakila tests on MySQL 8.4
   MYSQL_SAKILA_DSN="root:testpass@tcp(localhost:3307)/sakila?parseTime=true" \
     go test -tags=integration ./tests/integration -run Sakila -v

4) Start MySQL 9.0 from compose
   docker compose -f docker-compose.test.yml up -d mysql90

5) Ensure mysql90 is healthy (`mysql-mcp-test-90`)
   wait_docker_healthy mysql-mcp-test-90

6) Run Sakila tests on MySQL 9.0
   MYSQL_SAKILA_DSN="root:testpass@tcp(localhost:3308)/sakila?parseTime=true" \
     go test -tags=integration ./tests/integration -run Sakila -v

7) Start MariaDB 11.4 from compose
   docker compose -f docker-compose.test.yml up -d mariadb11

8) Ensure mariadb11 is healthy (`mysql-mcp-test-mariadb-11`)
   wait_docker_healthy mysql-mcp-test-mariadb-11

9) Run Sakila tests on MariaDB 11.4
   MYSQL_SAKILA_DSN="root:testpass@tcp(localhost:3310)/sakila?parseTime=true" \
     go test -tags=integration ./tests/integration -run Sakila -v

10) Run MySQL 8.0 on alternate port (3306 was in use)
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

11) Wait for mysql-mcp-test-80-alt to be ready (retry until mysqladmin ping succeeds)
    wait_mysqladmin_ping mysql-mcp-test-80-alt

12) Run Sakila tests on MySQL 8.0 (alt port)
    MYSQL_SAKILA_DSN="root:testpass@tcp(localhost:3311)/sakila?parseTime=true" \
      go test -tags=integration ./tests/integration -run Sakila -v

Cleanup (requested option 2)
13) Stop and remove compose containers, network, and volumes
    docker compose -f docker-compose.test.yml down -v

14) Remove the MySQL 8.0 alt container and volume
    docker rm -f mysql-mcp-test-80-alt
    docker volume rm mysql80_alt_data
