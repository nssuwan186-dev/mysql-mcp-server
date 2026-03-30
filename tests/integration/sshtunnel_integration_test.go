//go:build integration
// +build integration

package integration

import (
	"context"
	"database/sql"
	"os"
	"path/filepath"
	"strconv"
	"testing"
	"time"

	"github.com/askdba/mysql-mcp-server/internal/sshtunnel"
	"github.com/go-sql-driver/mysql"
	_ "github.com/go-sql-driver/mysql"
)

// TestSSHTunnel verifies that we can reach MySQL through an SSH bastion using
// internal/sshtunnel. Requires docker compose with mysql80 + ssh_bastion:
//
//	docker compose -f docker-compose.test.yml up -d mysql80 ssh_bastion
//
// Then run with:
//
//	MYSQL_SSH_HOST=localhost MYSQL_SSH_PORT=2222 MYSQL_SSH_USER=root \
//	MYSQL_SSH_KEY_PATH=$PWD/tests/integration/fixtures/ssh_test_key \
//	MYSQL_SSH_TEST_DSN="mcpuser:mcppass00@tcp(mysql80:3306)/testdb?parseTime=true" \
//	go test -tags=integration -v -run TestSSHTunnel ./tests/integration/...
func TestSSHTunnel(t *testing.T) {
	host := os.Getenv("MYSQL_SSH_HOST")
	keyPath := os.Getenv("MYSQL_SSH_KEY_PATH")
	dsnBehindBastion := os.Getenv("MYSQL_SSH_TEST_DSN")
	if host == "" || keyPath == "" || dsnBehindBastion == "" {
		t.Skip("MYSQL_SSH_HOST, MYSQL_SSH_KEY_PATH, and MYSQL_SSH_TEST_DSN required for SSH tunnel integration test")
	}
	user := os.Getenv("MYSQL_SSH_USER")
	if user == "" {
		user = "root"
	}
	port := 22
	if v := os.Getenv("MYSQL_SSH_PORT"); v != "" {
		if p, err := strconv.Atoi(v); err == nil {
			port = p
		}
	}

	// Resolve key path to absolute so it works from any cwd
	absKey, err := filepath.Abs(keyPath)
	if err != nil {
		t.Fatalf("resolve key path: %v", err)
	}
	if _, err := os.Stat(absKey); err != nil {
		t.Skipf("SSH key not found at %s (set MYSQL_SSH_KEY_PATH)", absKey)
	}

	cfg, err := mysql.ParseDSN(dsnBehindBastion)
	if err != nil {
		t.Fatalf("parse MYSQL_SSH_TEST_DSN: %v", err)
	}
	remoteAddr := cfg.Addr
	if remoteAddr == "" {
		remoteAddr = "mysql80:3306"
	}

	tunnelCfg := sshtunnel.Config{
		Host:    host,
		User:    user,
		KeyPath: absKey,
		Port:    port,
	}
	localAddr, closeTunnel, err := sshtunnel.Tunnel(tunnelCfg, remoteAddr)
	if err != nil {
		t.Fatalf("start SSH tunnel: %v", err)
	}
	defer closeTunnel()

	cfg.Addr = localAddr
	localDSN := cfg.FormatDSN()
	db, err := sql.Open("mysql", localDSN)
	if err != nil {
		t.Fatalf("open mysql through tunnel: %v", err)
	}
	defer db.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	if err := db.PingContext(ctx); err != nil {
		t.Fatalf("ping through tunnel: %v", err)
	}

	var one int
	if err := db.QueryRowContext(ctx, "SELECT 1").Scan(&one); err != nil {
		t.Fatalf("SELECT 1 through tunnel: %v", err)
	}
	if one != 1 {
		t.Errorf("SELECT 1: got %d, want 1", one)
	}

	// Verify we see the test database
	var dbName string
	if err := db.QueryRowContext(ctx, "SELECT DATABASE()").Scan(&dbName); err != nil {
		t.Fatalf("SELECT DATABASE(): %v", err)
	}
	if dbName != "testdb" {
		t.Errorf("DATABASE(): got %q, want testdb", dbName)
	}
}
