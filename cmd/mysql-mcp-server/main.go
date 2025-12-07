// cmd/mysql-mcp-server/main.go
package main

import (
	"context"
	"database/sql"
	"log"
	"os"
	"strings"
	"time"

	_ "github.com/go-sql-driver/mysql"
	"github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/askdba/mysql-mcp-server/internal/util"
)

// ===== Constants =====

const (
	defaultMaxRows            = 200
	defaultQueryTimeoutSecs   = 30
	defaultMaxOpenConns       = 10
	defaultMaxIdleConns       = 5
	defaultConnMaxLifetimeM   = 30
	defaultHTTPPort           = 9306
	defaultHTTPRequestTimeout = 60 * time.Second
)

// ===== Global State =====

// Global DB handle shared by all tools (safe for concurrent use).
var (
	db           *sql.DB
	maxRows      int
	queryTimeout time.Duration
	extendedMode bool
	jsonLogging  bool
	auditLogger  *AuditLogger
	connManager  *ConnectionManager
)

// ===== Utility Functions =====

func getEnvInt(key string, def int) int {
	val := strings.TrimSpace(os.Getenv(key))
	if val == "" {
		return def
	}
	var n int
	for _, c := range val {
		if c < '0' || c > '9' {
			return def
		}
		n = n*10 + int(c-'0')
	}
	if n <= 0 {
		return def
	}
	return n
}

// ===== Main Entry Point =====

func main() {
	// ---- Load config from env ----
	maxRows = getEnvInt("MYSQL_MAX_ROWS", defaultMaxRows)
	qTimeoutSecs := getEnvInt("MYSQL_QUERY_TIMEOUT_SECONDS", defaultQueryTimeoutSecs)
	queryTimeout = time.Duration(qTimeoutSecs) * time.Second

	// Check for extended mode
	extendedMode = os.Getenv("MYSQL_MCP_EXTENDED") == "1"

	// Check for JSON logging
	jsonLogging = os.Getenv("MYSQL_MCP_JSON_LOGS") == "1"

	// Check for vector tools (MySQL 9.0+)
	vectorMode := os.Getenv("MYSQL_MCP_VECTOR") == "1"

	// Check for HTTP REST API mode
	httpMode := os.Getenv("MYSQL_MCP_HTTP") == "1"
	httpPort := getEnvInt("MYSQL_HTTP_PORT", defaultHTTPPort)

	// Initialize audit logger if path is specified
	auditLogPath := strings.TrimSpace(os.Getenv("MYSQL_MCP_AUDIT_LOG"))
	var err error
	auditLogger, err = NewAuditLogger(auditLogPath)
	if err != nil {
		log.Fatalf("audit log init error: %v", err)
	}
	if auditLogger.enabled {
		defer auditLogger.Close()
	}

	// ---- Initialize Connection Manager ----
	connManager = NewConnectionManager()
	defer connManager.Close()

	// Load connections from environment
	connConfigs, err := loadConnectionsFromEnv()
	if err != nil {
		log.Fatalf("config error: %v", err)
	}

	if len(connConfigs) == 0 {
		log.Fatalf("config error: no MySQL connections configured. Set MYSQL_DSN or MYSQL_CONNECTIONS")
	}

	// Add all connections to the manager
	for _, cfg := range connConfigs {
		if err := connManager.AddConnection(cfg); err != nil {
			log.Printf("Warning: failed to add connection '%s': %v", cfg.Name, err)
		} else {
			logInfo("connection added", map[string]interface{}{
				"name": cfg.Name,
				"dsn":  util.MaskDSN(cfg.DSN),
			})
		}
	}

	// Set the global db to the active connection
	db = connManager.GetActiveDB()
	if db == nil {
		log.Fatalf("config error: no valid MySQL connections available")
	}

	_, activeName := connManager.GetActive()

	// Log startup configuration
	logInfo("mysql-mcp-server started", map[string]interface{}{
		"maxRows":          maxRows,
		"queryTimeout":     queryTimeout.String(),
		"extendedMode":     extendedMode,
		"vectorMode":       vectorMode,
		"httpMode":         httpMode,
		"httpPort":         httpPort,
		"jsonLogging":      jsonLogging,
		"auditLogEnabled":  auditLogger.enabled,
		"connections":      len(connConfigs),
		"activeConnection": activeName,
	})

	// If HTTP mode is enabled, start REST API server instead of MCP
	if httpMode {
		startHTTPServer(httpPort, vectorMode)
		return
	}

	// ---- Build MCP server ----
	server := mcp.NewServer(
		&mcp.Implementation{
			Name:    "mysql-mcp-server",
			Version: "1.1.0",
		},
		nil,
	)

	// Register core tools
	registerCoreTools(server)

	// Register multi-DSN tools
	registerConnectionTools(server)

	// Register vector tools (MYSQL_MCP_VECTOR=1)
	if vectorMode {
		registerVectorTools(server)
	}

	// Register extended tools (MYSQL_MCP_EXTENDED=1)
	if extendedMode {
		registerExtendedTools(server)
	}

	// ---- Run over stdio ----
	if err := server.Run(context.Background(), &mcp.StdioTransport{}); err != nil {
		log.Fatal(err)
	}
}

// ===== Tool Registration =====

func registerCoreTools(server *mcp.Server) {
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
}

func registerConnectionTools(server *mcp.Server) {
	mcp.AddTool(server, &mcp.Tool{
		Name:        "list_connections",
		Description: "List all configured MySQL connections and show which is active",
	}, toolListConnections)

	mcp.AddTool(server, &mcp.Tool{
		Name:        "use_connection",
		Description: "Switch to a different MySQL connection by name",
	}, toolUseConnection)
}

func registerVectorTools(server *mcp.Server) {
	logInfo("Registering MySQL vector tools (MySQL 9.0+ required)...", nil)

	mcp.AddTool(server, &mcp.Tool{
		Name:        "vector_search",
		Description: "Perform similarity search on vector columns (MySQL 9.0+ required)",
	}, toolVectorSearch)

	mcp.AddTool(server, &mcp.Tool{
		Name:        "vector_info",
		Description: "List vector columns and their properties in a database",
	}, toolVectorInfo)
}

func registerExtendedTools(server *mcp.Server) {
	log.Printf("Registering extended MySQL tools...")

	mcp.AddTool(server, &mcp.Tool{
		Name:        "list_indexes",
		Description: "List indexes on a table",
	}, toolListIndexes)

	mcp.AddTool(server, &mcp.Tool{
		Name:        "show_create_table",
		Description: "Show the CREATE TABLE statement for a table",
	}, toolShowCreateTable)

	mcp.AddTool(server, &mcp.Tool{
		Name:        "explain_query",
		Description: "Get the execution plan for a SELECT query",
	}, toolExplainQuery)

	mcp.AddTool(server, &mcp.Tool{
		Name:        "list_views",
		Description: "List views in a database",
	}, toolListViews)

	mcp.AddTool(server, &mcp.Tool{
		Name:        "list_triggers",
		Description: "List triggers in a database",
	}, toolListTriggers)

	mcp.AddTool(server, &mcp.Tool{
		Name:        "list_procedures",
		Description: "List stored procedures in a database",
	}, toolListProcedures)

	mcp.AddTool(server, &mcp.Tool{
		Name:        "list_functions",
		Description: "List stored functions in a database",
	}, toolListFunctions)

	mcp.AddTool(server, &mcp.Tool{
		Name:        "list_partitions",
		Description: "List partitions of a table",
	}, toolListPartitions)

	mcp.AddTool(server, &mcp.Tool{
		Name:        "database_size",
		Description: "Get size information for databases",
	}, toolDatabaseSize)

	mcp.AddTool(server, &mcp.Tool{
		Name:        "table_size",
		Description: "Get size information for tables",
	}, toolTableSize)

	mcp.AddTool(server, &mcp.Tool{
		Name:        "foreign_keys",
		Description: "List foreign key constraints",
	}, toolForeignKeys)

	mcp.AddTool(server, &mcp.Tool{
		Name:        "list_status",
		Description: "List MySQL server status variables",
	}, toolListStatus)

	mcp.AddTool(server, &mcp.Tool{
		Name:        "list_variables",
		Description: "List MySQL server configuration variables",
	}, toolListVariables)
}
