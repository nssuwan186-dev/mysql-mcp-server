MySQL MCP Server

A fast, read-only MySQL Server for the Model Context Protocol (MCP) written in Go.

This project exposes safe MySQL introspection tools to Claude Desktop via MCP. Claude can use these tools to explore databases, describe schemas, and run controlled read-only SQL queries — ideal for secure assistance during development, debugging, analytics, or schema documentation.

⸻

🚀 Features
	•	🔒 Fully read-only (blocks all non-SELECT/SHOW/DESCRIBE/EXPLAIN statements)
	•	🧰 MCP tools provided
	•	list_databases
	•	list_tables
	•	describe_table
	•	run_query (safe & row-limited)
	•	🐬 Compatible with MySQL 5.7 / 8.0 / 8.4
	•	⏱ Query timeout enforcement
	•	📦 Zero external runtime dependencies — single Go binary
	•	🧪 Unit + integration tests using Testcontainers
	•	🔌 Drop-in integration with Claude Desktop

⸻

📦 Installation

1. Clone + build

git clone https://github.com/askdba/mysql-mcp-server.git
cd mysql-mcp-server
make build

The binary will be created at:

bin/mysql-mcp-server


⸻

⚙️ Configuration

The server is configured using environment variables:

Variable	Required	Default	Description
MYSQL_DSN	✅ Yes	–	MySQL connection string (Go DSN format)
MYSQL_MAX_ROWS	No	200	Max rows returned by run_query
MYSQL_QUERY_TIMEOUT_SECONDS	No	30	Query timeout, in seconds
MYSQL_READONLY	No	1	Read-only enforcement

Example

export MYSQL_DSN="root:password@tcp(127.0.0.1:3306)/mysql?parseTime=true"
export MYSQL_MAX_ROWS=200
export MYSQL_QUERY_TIMEOUT_SECONDS=30

Run:

make run


⸻

🧠 Claude Desktop Integration

This server integrates directly with Claude Desktop via the official MCP interface.

1. Create (or edit) the Claude config file:

~/Library/Application Support/Claude/claude_desktop_config.json

Add:

{
  "mcpServers": {
    "mysql": {
      "command": "/absolute/path/to/bin/mysql-mcp-server",
      "env": {
        "MYSQL_DSN": "root:password@tcp(127.0.0.1:3306)/mysql?parseTime=true",
        "MYSQL_MAX_ROWS": "200"
      }
    }
  }
}

2. Restart Claude Desktop

Claude will automatically detect the MySQL MCP server and load the tools.

⸻

🧰 MCP Tools

The server exposes the following tools to Claude:

1. list_databases

Returns non-system databases.

⸻

2. list_tables

Input:

{
  "database": "employees"
}


⸻

3. describe_table

Input:

{
  "database": "employees",
  "table": "salaries"
}

Returns column names, types, nullability, and other metadata.

⸻

4. run_query

Input:

{
  "sql": "SELECT id, name FROM users LIMIT 5"
}

	•	Rejects any non-read-only SQL
	•	Enforces limit + timeout
	•	Never returns more than MYSQL_MAX_ROWS

⸻

🔒 Security Model

This server is explicitly built for safe AI-assisted database access:
	•	Read-only SQL enforcement
	•	Row limits to prevent dumping large tables
	•	Query timeout
	•	Recommended to use a dedicated low-privilege MySQL user:

CREATE USER 'mcp'@'localhost' IDENTIFIED BY 'strongpass';
GRANT SELECT ON *.* TO 'mcp'@'localhost';


⸻

🧪 Testing

Unit tests:

make test

Integration tests (requires Docker):

make integration

This spins up a temporary MySQL container using Testcontainers and runs real queries.

⸻

🐳 Docker

Build image:

docker build -t mysql-mcp-server .

Example docker-compose.yml:

version: "3.9"
services:
  mysql:
    image: mysql:8.0.36
    environment:
      MYSQL_ROOT_PASSWORD: rootpass
      MYSQL_DATABASE: testdb
    ports:
      - "3306:3306"

  mcp:
    build: .
    depends_on:
      - mysql
    environment:
      MYSQL_DSN: "root:rootpass@tcp(mysql:3306)/testdb?parseTime=true"

Run:

docker compose up --build


⸻

📁 Project Structure

cmd/mysql-mcp-server/   → Main MCP server entrypoint
internal/config/        → Configuration loader
internal/mysql/         → DB client + unit + integration tests
bin/                    → Built binaries


⸻

👨‍💻 Development

Format:

make fmt

Run:

make run

Build:

make build


⸻

🤝 Contributing

Contributions and PRs are welcome.

Possible areas:
	•	Extra MySQL MCP tools (list_indexes, explain_query, etc.)
	•	More schema exploration helpers
	•	Healthcheck tool
	•	Audit logging
	•	MySQL connection pooling configuration

⸻

📜 License

Apache License 2.0
© 2025 Alkin Tezuysal

