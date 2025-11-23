package config

import (
	"os"
	"testing"
)

func TestLoadWithDefaults(t *testing.T) {
	// Clean env
	_ = os.Unsetenv("MYSQL_DSN")
	_ = os.Unsetenv("MYSQL_MCP_MAX_ROWS")
	_ = os.Unsetenv("MYSQL_MCP_QUERY_TIMEOUT_S")

	// Required DSN
	if err := os.Setenv("MYSQL_DSN", "user:pass@tcp(localhost:3306)/testdb"); err != nil {
		t.Fatalf("failed to set env: %v", err)
	}

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load returned error: %v", err)
	}

	if cfg.DSN == "" {
		t.Fatalf("expected DSN to be set")
	}

	if cfg.MaxRows != 200 {
		t.Fatalf("expected default MaxRows=200, got %d", cfg.MaxRows)
	}

	if cfg.QueryTimeoutS != 10 {
		t.Fatalf("expected default QueryTimeoutS=10, got %d", cfg.QueryTimeoutS)
	}
}

func TestLoadOverridesFromEnv(t *testing.T) {
	if err := os.Setenv("MYSQL_DSN", "user:pass@tcp(localhost:3306)/otherdb"); err != nil {
		t.Fatalf("failed to set MYSQL_DSN: %v", err)
	}
	if err := os.Setenv("MYSQL_MCP_MAX_ROWS", "500"); err != nil {
		t.Fatalf("failed to set MYSQL_MCP_MAX_ROWS: %v", err)
	}
	if err := os.Setenv("MYSQL_MCP_QUERY_TIMEOUT_S", "30"); err != nil {
		t.Fatalf("failed to set MYSQL_MCP_QUERY_TIMEOUT_S: %v", err)
	}

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load returned error: %v", err)
	}

	if cfg.DSN != "user:pass@tcp(localhost:3306)/otherdb" {
		t.Fatalf("unexpected DSN: %s", cfg.DSN)
	}
	if cfg.MaxRows != 500 {
		t.Fatalf("expected MaxRows=500, got %d", cfg.MaxRows)
	}
	if cfg.QueryTimeoutS != 30 {
		t.Fatalf("expected QueryTimeoutS=30, got %d", cfg.QueryTimeoutS)
	}
}

func TestLoadMissingDSN(t *testing.T) {
	_ = os.Unsetenv("MYSQL_DSN")

	_, err := Load()
	if err == nil {
		t.Fatalf("expected error when MYSQL_DSN is missing")
	}
}
