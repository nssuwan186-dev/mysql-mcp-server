# Silent and daemon mode

This document describes how to run mysql-mcp-server with minimal console output (silent mode) and as a background process (daemon mode), suitable for production and service managers.

## Silent mode (`-s` / `--silent`)

When `--silent` (or `-s`) is set:

- **INFO** and **WARN** log messages are suppressed.
- **ERROR** messages are still written to stderr so failures and diagnostics remain visible.

Use silent mode when:

- Running under systemd, launchd, or another process manager that logs separately.
- Reducing noise in container or CI logs.
- The server is already monitored via health checks or audit logs.

**Examples:**

```bash
mysql-mcp-server --silent --config /etc/mysql-mcp-server/config.yaml
MYSQL_MCP_HTTP=1 mysql-mcp-server --silent --config /path/to/config.yaml
```

Structured (JSON) logging is unaffected by `--silent` for the **level** of messages: if `MYSQL_MCP_JSON_LOGS=1`, only INFO/WARN lines are skipped; ERROR lines are still emitted as JSON.

## Daemon mode (`-d` / `--daemon`)

When `--daemon` (or `-d`) is set on **Unix**:

- The process **forks** a child that runs the server.
- The **parent** exits immediately (exit code 0).
- The **child** runs in a new session (detached from the terminal) and continues with the same configuration.

Daemon mode is intended for **HTTP REST API mode** (`MYSQL_MCP_HTTP=1`). Using it with stdio MCP is possible but less common (the child would read from stdin and write to stdout with no terminal).

**Examples:**

```bash
# Start HTTP server in background
MYSQL_MCP_HTTP=1 mysql-mcp-server --daemon --config /path/to/config.yaml

# With silent mode (recommended when running as daemon)
MYSQL_MCP_HTTP=1 mysql-mcp-server --daemon --silent --config /path/to/config.yaml
```

**Windows:** `--daemon` is a no-op. Use a Windows Service or run the process in the background (e.g. in a separate terminal or via a scheduler).

## Service manager templates

The repository includes example unit files for running the server under a process manager:

| Platform   | File                                                                 | Description                    |
|-----------|----------------------------------------------------------------------|--------------------------------|
| systemd   | [contrib/systemd/mysql-mcp-server.service](contrib/systemd/mysql-mcp-server.service) | Example systemd unit           |
| launchd   | [contrib/launchd/com.askdba.mysql-mcp-server.plist](contrib/launchd/com.askdba.mysql-mcp-server.plist) | Example launchd plist (macOS)  |

### systemd (Linux)

1. Copy the unit file:  
   `sudo cp contrib/systemd/mysql-mcp-server.service /etc/systemd/system/`
2. Edit paths and environment:  
   `sudo edit /etc/systemd/system/mysql-mcp-server.service`  
   Set `ExecStart` to your binary and config path. Optionally use `EnvironmentFile=` for `MYSQL_*` variables.
3. Reload and enable:  
   `sudo systemctl daemon-reload`  
   `sudo systemctl enable --now mysql-mcp-server`

### launchd (macOS)

1. Copy the plist:  
   `cp contrib/launchd/com.askdba.mysql-mcp-server.plist ~/Library/LaunchAgents/`  
   (or `/Library/LaunchDaemons/` for a system-wide service)
2. Edit `ProgramArguments` and paths (binary, config, log paths).
3. Load and start:  
   `launchctl load ~/Library/LaunchAgents/com.askdba.mysql-mcp-server.plist`

## Summary

| Flag / option   | Effect |
|-----------------|--------|
| `-s` / `--silent` | Suppress INFO and WARN; ERROR still to stderr |
| `-d` / `--daemon` | Fork and detach (Unix); parent exits, child runs server |
| contrib/systemd  | Example systemd unit for Linux |
| contrib/launchd  | Example launchd plist for macOS |

For full CLI options, run `mysql-mcp-server --help`.
