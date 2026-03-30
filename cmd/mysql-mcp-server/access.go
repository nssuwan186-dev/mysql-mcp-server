package main

import (
	"fmt"
	"strings"

	"github.com/askdba/mysql-mcp-server/internal/config"
)

var allowedDatabaseSet map[string]struct{}

func initAccessControl(allowed []string) {
	allowedDatabaseSet = config.AllowedDatabaseSet(allowed)
}

func accessControlEnabled() bool {
	return len(allowedDatabaseSet) > 0
}

func databaseAllowed(name string) bool {
	if !accessControlEnabled() {
		return true
	}
	name = strings.TrimSpace(name)
	if name == "" {
		return false
	}
	_, ok := allowedDatabaseSet[strings.ToLower(name)]
	return ok
}

func requireAllowedDatabase(db string) error {
	if !accessControlEnabled() {
		return nil
	}
	if strings.TrimSpace(db) == "" {
		return fmt.Errorf("database is required when MYSQL_MCP_ALLOWED_DATABASES is configured")
	}
	if !databaseAllowed(db) {
		return fmt.Errorf("database %q is not in MYSQL_MCP_ALLOWED_DATABASES", db)
	}
	return nil
}
