package main

import (
	"fmt"
	"sort"
	"strings"

	"github.com/askdba/mysql-mcp-server/internal/config"
	"github.com/askdba/mysql-mcp-server/internal/util"
)

var allowedDatabaseSet map[string]struct{}

func initAccessControl(allowed []string) {
	allowedDatabaseSet = config.AllowedDatabaseSet(allowed)
}

func accessControlEnabled() bool {
	return len(allowedDatabaseSet) > 0
}

// allowedDatabasesLower returns allowlist entries as lowercase strings, sorted.
// Used for SQL filters (e.g. mysql.slow_log.db) when accessControlEnabled().
func allowedDatabasesLower() []string {
	if !accessControlEnabled() {
		return nil
	}
	out := make([]string, 0, len(allowedDatabaseSet))
	for name := range allowedDatabaseSet {
		out = append(out, name)
	}
	sort.Strings(out)
	return out
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

// requireReferencedSchemasInQuery ensures every explicitly schema-qualified
// reference in sqlText (and USE / SHOW / EXPLAIN targets) is allowed when an
// allowlist is configured.
func requireReferencedSchemasInQuery(sqlText string) error {
	if !accessControlEnabled() {
		return nil
	}
	refs, err := util.ReferencedSchemaQualifiers(sqlText)
	if err != nil {
		return fmt.Errorf("query validation failed: %w", err)
	}
	for name := range refs {
		if !databaseAllowed(name) {
			return fmt.Errorf("query references database %q which is not in MYSQL_MCP_ALLOWED_DATABASES", name)
		}
	}
	return nil
}
