# Production usage examples

Example invocations for running mysql-mcp-server in production or under a process manager.

## Silent mode (minimal logs)

Only errors are written to stderr; INFO and WARN are suppressed.

```bash
mysql-mcp-server --silent --config /etc/mysql-mcp-server/config.yaml
```

With environment-only config:

```bash
export MYSQL_DSN="user:pass@tcp(host:3306)/db?parseTime=true"
mysql-mcp-server --silent
```

## HTTP server as daemon (Unix)

Fork and run the REST API server in the background. The parent process exits; the child runs detached.

```bash
export MYSQL_DSN="user:pass@tcp(host:3306)/db?parseTime=true"
export MYSQL_MCP_HTTP=1
export MYSQL_HTTP_PORT=9306
mysql-mcp-server --daemon --config /path/to/config.yaml
```

With silent mode (recommended when running as daemon):

```bash
MYSQL_MCP_HTTP=1 mysql-mcp-server --daemon --silent --config /path/to/config.yaml
```

## systemd (Linux)

See [contrib/systemd/mysql-mcp-server.service](../contrib/systemd/mysql-mcp-server.service). After copying and editing:

```bash
sudo systemctl daemon-reload
sudo systemctl enable --now mysql-mcp-server
```

## launchd (macOS)

See [contrib/launchd/com.askdba.mysql-mcp-server.plist](../contrib/launchd/com.askdba.mysql-mcp-server.plist). After copying and editing:

```bash
launchctl load ~/Library/LaunchAgents/com.askdba.mysql-mcp-server.plist
```

## More details

- [Silent and daemon mode](../docs/silent-and-daemon.md) in the docs folder.
- [README configuration section](../README.md#configuration) for config file and env vars.
