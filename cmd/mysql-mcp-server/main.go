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
	db          *sql.DB
	maxRows     int
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
	Name     string `json:"name" jsonschema:"column name"`
	Type     string `json:"type" jsonschema:"column type"`
	Null     string `json:"null" jsonschema:"YES if nullable, NO otherwise"`
	Key      string `json:"key" jsonschema:"key information (PRI, MUL, etc.)"`
	Default  string `json:"default" jsonschema:"default value, if any"`
	Extra    string `json:"extra" jsonschema:"extra metadata (auto_increment, etc.)"`
	Comment  string `json:"comment" jsonschema:"column comment, if any"`
	Collation string `json:"collation" jsonschema:"column collation, if any"`
}

type DescribeTableOutput struct {
	Columns []ColumnInfo `json:"columns" jsonschema:"detailed column information"`
}

type RunQueryInput struct {
	SQL      string `json:"sql" jsonschema:"SQL query to execute; must start with SELECT, SHOW, DESCRIBE, or EXPLAIN"`
	MaxRows  *int   `json:"max_rows,omitempty" jsonschema:"optional row limit overriding the default max rows"`
	Database string `json:"database,omitempty" jsonschema:"optional database name (currently informational only)"`
}

type QueryResult struct {
	Columns []string        `json:"columns" jsonschema:"column names"`
	Rows    [][]interface{} `json:"rows" jsonschema:"rows of values"`
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

func quoteIdent(name string) string {
	// Very simple identifier quoting. We also do a basic sanity check
	// to avoid obviously dangerous values.
	if name == "" {
		return ""
	}
	if strings.ContainsAny(name, " \t\n\r;`") {
		// fall back to plain; better to error earlier in callers if needed
		return name
	}
	return "`" + name + "`"
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

	dbName := quoteIdent(input.Database)
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

	dbName := quoteIdent(input.Database)
	tableName := quoteIdent(input.Table)

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

	// ---- Run over stdio ----
	if err := server.Run(context.Background(), &mcp.StdioTransport{}); err != nil {
		log.Fatal(err)
	}
}
