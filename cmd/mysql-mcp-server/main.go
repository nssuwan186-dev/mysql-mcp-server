// cmd/mysql-mcp-server/main.go
package main

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"os"
	"strconv"
	"strings"
	"time"

	_ "github.com/go-sql-driver/mysql"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

const (
	defaultMaxRows          = 200
	defaultQueryTimeoutSecs = 30
)

// Global DB handle shared by all tools (safe for concurrent use).
var (
	db           *sql.DB
	maxRows      int
	queryTimeout time.Duration
)

// ===== Tool input / output types =====

type ListDatabasesInput struct{}

type DatabaseInfo struct {
	Name string `json:"name" jsonschema:"database name"`
}

type ListDatabasesOutput struct {
	Databases []DatabaseInfo `json:"databases" jsonschema:"list of accessible databases"`
}

type ListTablesInput struct {
	Database string `json:"database" jsonschema:"database name to list tables from"`
}

type TableInfo struct {
	Name string `json:"name" jsonschema:"table name"`
}

type ListTablesOutput struct {
	Tables []TableInfo `json:"tables" jsonschema:"list of tables in the database"`
}

type DescribeTableInput struct {
	Database string `json:"database" jsonschema:"database name"`
	Table    string `json:"table" jsonschema:"table name"`
}

type ColumnInfo struct {
	Name      string `json:"name" jsonschema:"column name"`
	Type      string `json:"type" jsonschema:"column type"`
	Null      string `json:"null" jsonschema:"YES if nullable, NO otherwise"`
	Key       string `json:"key" jsonschema:"key information (PRI, MUL, etc.)"`
	Default   string `json:"default" jsonschema:"default value, if any"`
	Extra     string `json:"extra" jsonschema:"extra metadata (auto_increment, etc.)"`
	Comment   string `json:"comment" jsonschema:"column comment, if any"`
	Collation string `json:"collation" jsonschema:"column collation, if any"`
}

type DescribeTableOutput struct {
	Columns []ColumnInfo `json:"columns" jsonschema:"detailed column information"`
}

type RunQueryInput struct {
	SQL      string `json:"sql" jsonschema:"SQL query to execute; must start with SELECT, SHOW, DESCRIBE, or EXPLAIN"`
	MaxRows  *int   `json:"max_rows,omitempty" jsonschema:"optional row limit overriding the default max rows"`
	Database string `json:"database,omitempty" jsonschema:"optional database name to USE before running the query"`
}

type QueryResult struct {
	Columns []string        `json:"columns" jsonschema:"column names"`
	Rows    [][]interface{} `json:"rows" jsonschema:"rows of values"`
}

type PingInput struct{}

type PingOutput struct {
	Success   bool   `json:"success" jsonschema:"true if the database is reachable"`
	LatencyMs int64  `json:"latency_ms" jsonschema:"round-trip latency in milliseconds"`
	Message   string `json:"message" jsonschema:"status message"`
}

type ServerInfoInput struct{}

type ServerInfoOutput struct {
	Version         string `json:"version" jsonschema:"MySQL server version"`
	VersionComment  string `json:"version_comment" jsonschema:"MySQL version comment (e.g., MySQL Community Server)"`
	Uptime          int64  `json:"uptime_seconds" jsonschema:"server uptime in seconds"`
	CurrentDatabase string `json:"current_database" jsonschema:"currently selected database, if any"`
	CurrentUser     string `json:"current_user" jsonschema:"current MySQL user"`
	CharacterSet    string `json:"character_set" jsonschema:"server character set"`
	Collation       string `json:"collation" jsonschema:"server collation"`
	MaxConnections  int    `json:"max_connections" jsonschema:"maximum allowed connections"`
	ThreadsConnected int   `json:"threads_connected" jsonschema:"current number of connected threads"`
}

// ===== Utility: env config & helpers =====

func getEnvInt(key string, def int) int {
	val := strings.TrimSpace(os.Getenv(key))
	if val == "" {
		return def
	}
	n, err := strconv.Atoi(val)
	if err != nil || n <= 0 {
		return def
	}
	return n
}

// quoteIdent safely quotes a MySQL identifier, returning an error if the name
// contains potentially dangerous characters.
func quoteIdent(name string) (string, error) {
	if name == "" {
		return "", fmt.Errorf("identifier cannot be empty")
	}
	// Reject identifiers with dangerous characters that could enable SQL injection
	if strings.ContainsAny(name, " \t\n\r;`\\") {
		return "", fmt.Errorf("identifier contains invalid characters: %q", name)
	}
	// Additional check: reject identifiers that are too long (MySQL limit is 64)
	if len(name) > 64 {
		return "", fmt.Errorf("identifier too long: %d characters (max 64)", len(name))
	}
	return "`" + name + "`", nil
}

// convert raw DB value into something JSON-friendly.
func normalizeValue(v interface{}) interface{} {
	switch x := v.(type) {
	case nil:
		return nil
	case []byte:
		return string(x)
	default:
		return x
	}
}

// Basic read-only safety check for run_query.
func isReadOnlySQL(sqlText string) bool {
	s := strings.TrimSpace(sqlText)
	if s == "" {
		return false
	}
	upper := strings.ToUpper(s)
	return strings.HasPrefix(upper, "SELECT") ||
		StringsHasPrefixAny(upper, []string{"SHOW", "DESCRIBE", "DESC", "EXPLAIN"})
}

func StringsHasPrefixAny(s string, prefixes []string) bool {
	for _, p := range prefixes {
		if strings.HasPrefix(s, p) {
			return true
		}
	}
	return false
}

// ===== Tool handlers =====

func toolListDatabases(
	ctx context.Context,
	req *mcp.CallToolRequest,
	input ListDatabasesInput,
) (*mcp.CallToolResult, ListDatabasesOutput, error) {

	ctx, cancel := context.WithTimeout(ctx, queryTimeout)
	defer cancel()

	rows, err := db.QueryContext(ctx, "SHOW DATABASES")
	if err != nil {
		return nil, ListDatabasesOutput{}, fmt.Errorf("SHOW DATABASES failed: %w", err)
	}
	defer rows.Close()

	out := ListDatabasesOutput{Databases: []DatabaseInfo{}}
	for rows.Next() {
		var name string
		if err := rows.Scan(&name); err != nil {
			return nil, ListDatabasesOutput{}, fmt.Errorf("scan database name failed: %w", err)
		}
		out.Databases = append(out.Databases, DatabaseInfo{Name: name})
		if len(out.Databases) >= maxRows {
			break
		}
	}
	if err := rows.Err(); err != nil {
		return nil, ListDatabasesOutput{}, err
	}

	return nil, out, nil
}

func toolListTables(
	ctx context.Context,
	req *mcp.CallToolRequest,
	input ListTablesInput,
) (*mcp.CallToolResult, ListTablesOutput, error) {

	if strings.TrimSpace(input.Database) == "" {
		return nil, ListTablesOutput{}, fmt.Errorf("database is required")
	}

	ctx, cancel := context.WithTimeout(ctx, queryTimeout)
	defer cancel()

	dbName, err := quoteIdent(input.Database)
	if err != nil {
		return nil, ListTablesOutput{}, fmt.Errorf("invalid database name: %w", err)
	}
	query := fmt.Sprintf("SHOW TABLES FROM %s", dbName)

	rows, err := db.QueryContext(ctx, query)
	if err != nil {
		return nil, ListTablesOutput{}, fmt.Errorf("SHOW TABLES failed: %w", err)
	}
	defer rows.Close()

	out := ListTablesOutput{Tables: []TableInfo{}}
	for rows.Next() {
		var name string
		if err := rows.Scan(&name); err != nil {
			return nil, ListTablesOutput{}, fmt.Errorf("scan table name failed: %w", err)
		}
		out.Tables = append(out.Tables, TableInfo{Name: name})
		if len(out.Tables) >= maxRows {
			break
		}
	}
	if err := rows.Err(); err != nil {
		return nil, ListTablesOutput{}, err
	}

	return nil, out, nil
}

func toolDescribeTable(
	ctx context.Context,
	req *mcp.CallToolRequest,
	input DescribeTableInput,
) (*mcp.CallToolResult, DescribeTableOutput, error) {

	if strings.TrimSpace(input.Database) == "" {
		return nil, DescribeTableOutput{}, fmt.Errorf("database is required")
	}
	if strings.TrimSpace(input.Table) == "" {
		return nil, DescribeTableOutput{}, fmt.Errorf("table is required")
	}

	ctx, cancel := context.WithTimeout(ctx, queryTimeout)
	defer cancel()

	dbName, err := quoteIdent(input.Database)
	if err != nil {
		return nil, DescribeTableOutput{}, fmt.Errorf("invalid database name: %w", err)
	}
	tableName, err := quoteIdent(input.Table)
	if err != nil {
		return nil, DescribeTableOutput{}, fmt.Errorf("invalid table name: %w", err)
	}

	// Using SHOW FULL COLUMNS to get richer metadata.
	query := fmt.Sprintf("SHOW FULL COLUMNS FROM %s.%s", dbName, tableName)

	rows, err := db.QueryContext(ctx, query)
	if err != nil {
		return nil, DescribeTableOutput{}, fmt.Errorf("SHOW FULL COLUMNS failed: %w", err)
	}
	defer rows.Close()

	out := DescribeTableOutput{Columns: []ColumnInfo{}}
	for rows.Next() {
		var col ColumnInfo
		var dummyPrivileges string

		// SHOW FULL COLUMNS FROM db.table returns:
		// Field, Type, Collation, Null, Key, Default, Extra, Privileges, Comment
		if err := rows.Scan(
			&col.Name,
			&col.Type,
			&col.Collation,
			&col.Null,
			&col.Key,
			&col.Default,
			&col.Extra,
			&dummyPrivileges,
			&col.Comment,
		); err != nil {
			return nil, DescribeTableOutput{}, fmt.Errorf("scan column failed: %w", err)
		}
		out.Columns = append(out.Columns, col)
		if len(out.Columns) >= maxRows {
			break
		}
	}
	if err := rows.Err(); err != nil {
		return nil, DescribeTableOutput{}, err
	}

	return nil, out, nil
}

func toolRunQuery(
	ctx context.Context,
	req *mcp.CallToolRequest,
	input RunQueryInput,
) (*mcp.CallToolResult, QueryResult, error) {

	sqlText := strings.TrimSpace(input.SQL)
	if sqlText == "" {
		return nil, QueryResult{}, fmt.Errorf("sql is required")
	}
	if !isReadOnlySQL(sqlText) {
		return nil, QueryResult{}, fmt.Errorf("only read-only queries are allowed (SELECT/SHOW/DESCRIBE/EXPLAIN)")
	}

	limit := maxRows
	if input.MaxRows != nil && *input.MaxRows > 0 && *input.MaxRows < maxRows {
		limit = *input.MaxRows
	}

	ctx, cancel := context.WithTimeout(ctx, queryTimeout)
	defer cancel()

	// Switch to the specified database if provided
	if database := strings.TrimSpace(input.Database); database != "" {
		dbName, err := quoteIdent(database)
		if err != nil {
			return nil, QueryResult{}, fmt.Errorf("invalid database name: %w", err)
		}
		if _, err := db.ExecContext(ctx, "USE "+dbName); err != nil {
			return nil, QueryResult{}, fmt.Errorf("failed to switch database: %w", err)
		}
	}

	rows, err := db.QueryContext(ctx, sqlText)
	if err != nil {
		return nil, QueryResult{}, fmt.Errorf("query failed: %w", err)
	}
	defer rows.Close()

	cols, err := rows.Columns()
	if err != nil {
		return nil, QueryResult{}, fmt.Errorf("get columns failed: %w", err)
	}

	result := QueryResult{
		Columns: cols,
		Rows:    make([][]interface{}, 0),
	}

	for rows.Next() {
		raw := make([]interface{}, len(cols))
		dest := make([]interface{}, len(cols))
		for i := range raw {
			dest[i] = &raw[i]
		}

		if err := rows.Scan(dest...); err != nil {
			return nil, QueryResult{}, fmt.Errorf("scan row failed: %w", err)
		}

		rowVals := make([]interface{}, len(cols))
		for i, v := range raw {
			rowVals[i] = normalizeValue(v)
		}
		result.Rows = append(result.Rows, rowVals)

		if len(result.Rows) >= limit {
			break
		}
	}
	if err := rows.Err(); err != nil {
		return nil, QueryResult{}, err
	}

	return nil, result, nil
}

func toolPing(
	ctx context.Context,
	req *mcp.CallToolRequest,
	input PingInput,
) (*mcp.CallToolResult, PingOutput, error) {

	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	start := time.Now()
	err := db.PingContext(ctx)
	latency := time.Since(start).Milliseconds()

	if err != nil {
		return nil, PingOutput{
			Success:   false,
			LatencyMs: latency,
			Message:   fmt.Sprintf("ping failed: %v", err),
		}, nil
	}

	return nil, PingOutput{
		Success:   true,
		LatencyMs: latency,
		Message:   "pong",
	}, nil
}

func toolServerInfo(
	ctx context.Context,
	req *mcp.CallToolRequest,
	input ServerInfoInput,
) (*mcp.CallToolResult, ServerInfoOutput, error) {

	ctx, cancel := context.WithTimeout(ctx, queryTimeout)
	defer cancel()

	out := ServerInfoOutput{}

	// Get version and version comment
	row := db.QueryRowContext(ctx, "SELECT VERSION()")
	if err := row.Scan(&out.Version); err != nil {
		return nil, ServerInfoOutput{}, fmt.Errorf("failed to get version: %w", err)
	}

	// Get various server variables in one query
	rows, err := db.QueryContext(ctx, `
		SELECT VARIABLE_NAME, VARIABLE_VALUE 
		FROM performance_schema.global_variables 
		WHERE VARIABLE_NAME IN (
			'version_comment', 
			'character_set_server', 
			'collation_server', 
			'max_connections'
		)
	`)
	if err != nil {
		// Fallback for older MySQL or restricted permissions
		rows, err = db.QueryContext(ctx, `
			SHOW VARIABLES WHERE Variable_name IN (
				'version_comment', 
				'character_set_server', 
				'collation_server', 
				'max_connections'
			)
		`)
		if err != nil {
			return nil, ServerInfoOutput{}, fmt.Errorf("failed to get server variables: %w", err)
		}
	}
	defer rows.Close()

	for rows.Next() {
		var name, value string
		if err := rows.Scan(&name, &value); err != nil {
			continue
		}
		switch strings.ToLower(name) {
		case "version_comment":
			out.VersionComment = value
		case "character_set_server":
			out.CharacterSet = value
		case "collation_server":
			out.Collation = value
		case "max_connections":
			out.MaxConnections, _ = strconv.Atoi(value)
		}
	}

	// Get uptime and threads connected from status
	statusRows, err := db.QueryContext(ctx, `
		SELECT VARIABLE_NAME, VARIABLE_VALUE 
		FROM performance_schema.global_status 
		WHERE VARIABLE_NAME IN ('Uptime', 'Threads_connected')
	`)
	if err != nil {
		// Fallback for older MySQL or restricted permissions
		statusRows, err = db.QueryContext(ctx, `
			SHOW GLOBAL STATUS WHERE Variable_name IN ('Uptime', 'Threads_connected')
		`)
		if err != nil {
			return nil, ServerInfoOutput{}, fmt.Errorf("failed to get server status: %w", err)
		}
	}
	defer statusRows.Close()

	for statusRows.Next() {
		var name, value string
		if err := statusRows.Scan(&name, &value); err != nil {
			continue
		}
		switch strings.ToLower(name) {
		case "uptime":
			out.Uptime, _ = strconv.ParseInt(value, 10, 64)
		case "threads_connected":
			out.ThreadsConnected, _ = strconv.Atoi(value)
		}
	}

	// Get current user and database
	row = db.QueryRowContext(ctx, "SELECT CURRENT_USER(), IFNULL(DATABASE(), '')")
	if err := row.Scan(&out.CurrentUser, &out.CurrentDatabase); err != nil {
		return nil, ServerInfoOutput{}, fmt.Errorf("failed to get current user/database: %w", err)
	}

	return nil, out, nil
}

// ===== main =====

func main() {
	// ---- Load config from env ----
	dsn := strings.TrimSpace(os.Getenv("MYSQL_DSN"))
	if dsn == "" {
		log.Fatalf("config error: MYSQL_DSN env var is required, e.g. user:pass@tcp(127.0.0.1:3306)/dbname?parseTime=true")
	}

	maxRows = getEnvInt("MYSQL_MAX_ROWS", defaultMaxRows)
	qTimeoutSecs := getEnvInt("MYSQL_QUERY_TIMEOUT_SECONDS", defaultQueryTimeoutSecs)
	queryTimeout = time.Duration(qTimeoutSecs) * time.Second

	// ---- Init DB ----
	var err error
	db, err = sql.Open("mysql", dsn)
	if err != nil {
		log.Fatalf("db init error: %v", err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := db.PingContext(ctx); err != nil {
		log.Fatalf("db init error: %v", err)
	}

	log.Printf("mysql-mcp-server connected to MySQL; maxRows=%d, queryTimeout=%s", maxRows, queryTimeout)

	// ---- Build MCP server ----
	server := mcp.NewServer(
		&mcp.Implementation{
			Name:    "mysql-mcp-server",
			Version: "0.1.0",
		},
		nil,
	)

	// Register tools.
	mcp.AddTool(server, &mcp.Tool{
		Name:        "list_databases",
		Description: "List accessible databases in the configured MySQL server",
	}, toolListDatabases)

	mcp.AddTool(server, &mcp.Tool{
		Name:        "list_tables",
		Description: "List tables in a given database",
	}, toolListTables)

	mcp.AddTool(server, &mcp.Tool{
		Name:        "describe_table",
		Description: "Describe columns of a given table",
	}, toolDescribeTable)

	mcp.AddTool(server, &mcp.Tool{
		Name:        "run_query",
		Description: "Execute a read-only SQL query (SELECT/SHOW/DESCRIBE/EXPLAIN only)",
	}, toolRunQuery)

	mcp.AddTool(server, &mcp.Tool{
		Name:        "ping",
		Description: "Test database connectivity and measure latency",
	}, toolPing)

	mcp.AddTool(server, &mcp.Tool{
		Name:        "server_info",
		Description: "Get MySQL server version, uptime, and configuration details",
	}, toolServerInfo)

	// ---- Run over stdio ----
	if err := server.Run(context.Background(), &mcp.StdioTransport{}); err != nil {
		log.Fatal(err)
	}
}
