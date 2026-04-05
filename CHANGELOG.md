# Changelog

All notable changes to this project will be documented in this file.

The format is based on "Keep a Changelog" and this project follows
Semantic Versioning.

## [Unreleased]

### Security

- **SSH bastion host keys**: the tunnel now verifies the server host key by default using OpenSSH-style **`known_hosts`** (default file `~/.ssh/known_hosts`, or **`MYSQL_SSH_KNOWN_HOSTS`** / config **`known_hosts`**) or a pinned fingerprint (**`MYSQL_SSH_HOST_KEY_FINGERPRINT`** / **`host_key_fingerprint`**). To disable verification (MITM risk), you must **opt in** with **`MYSQL_SSH_STRICT_HOST_KEY_CHECKING=false`** or **`ssh_strict_host_key_checking: false`**. See README.

### Added
- **`search_schema`**: Find tables and columns matching a pattern across all accessible databases.
- **`schema_diff`**: Compare table and column structures between two databases.
- **Column Masking**: Redact sensitive data in `run_query` results using **`MYSQL_MCP_MASK_COLUMNS`** (e.g., `email,password,token`).
- **Reliability**: Exponential-backoff retries for transient MySQL/network errors (deadlocks, timeouts, connection loss).

## [1.7.0-rc.3] - 2026-03-31

Third release candidate: metrics HTTP sidecar for stdio MCP (Claude Desktop) and friendlier boolean env parsing.

### Added

- **`MYSQL_MCP_METRICS_HTTP`**: optional HTTP listener on **`MYSQL_HTTP_PORT`** while MCP uses **stdio** — **`GET /health`**, **`GET /api/metrics/tokens`**, **`GET /status`** in-process with the MCP server so token metrics match Claude Desktop usage ([#102](https://github.com/askdba/mysql-mcp-server/issues/102)).
- **SSH tunneling (bastion host)**: connect to MySQL via `ssh_host`, `ssh_user`, `ssh_key_path`, and optional `ssh_port` (config file or `MYSQL_SSH_*` env vars). `key_path` supports `~` and `~/path` (expanded to user home). In this release, host key verification was not yet enforced; strict verification is documented under **[Unreleased]** ([#79](https://github.com/askdba/mysql-mcp-server/issues/79)).

### Changed

- **`getEnvBool`**: treats **`true`**, **`yes`**, **`on`**, **`y`** as true (case-insensitive), not only **`1`**, for **`MYSQL_MCP_*`** and related flags.
- **Full REST vs sidecar**: when **`MYSQL_MCP_HTTP=1`**, **`MetricsHTTP`** is cleared so the metrics-only listener does not run alongside the full HTTP API.

---

## [1.7.0-rc.2] - 2026-03-30
... rest of the file ...
