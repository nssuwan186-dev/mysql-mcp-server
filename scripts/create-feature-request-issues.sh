#!/usr/bin/env bash
# Creates GitHub issues from the 2026-03-30 feature / reliability backlog.
#
# Prerequisites:
#   gh auth login -h github.com
#
# Usage:
#   ./scripts/create-feature-request-issues.sh
#
# Set DRY_RUN=1 to print commands without executing.

set -euo pipefail

REPO="${REPO:-askdba/mysql-mcp-server}"
DRY_RUN="${DRY_RUN:-0}"

create_issue() {
	local title=$1
	shift
	local body=$1

	if [[ "$DRY_RUN" == "1" ]]; then
		echo "---- would create: $title ----"
		echo "$body"
		echo
		return
	fi
	gh issue create -R "$REPO" --title "$title" --body "$body"
}

# --- Missing tools / features ---

create_issue "Feature: write_query tool with explicit confirmation for INSERT/UPDATE/DELETE" "$(cat <<'EOF'
## Summary
Today the server is read-only; `run_query` only allows SELECT/SHOW/DESCRIBE/EXPLAIN. Agentic workflows often need safe, audited writes.

## Proposal
- Add `write_query` (or `execute_mutation`) that only accepts DML (INSERT/UPDATE/DELETE) or explicit allow-listed statements.
- Require a **confirmation** parameter (e.g. `confirm_token` matching a server-generated nonce, or a boolean `i_confirm=true` documented as dangerous) so hosts can implement UI consent.
- Enforce row limits / single-statement validation; optional dry-run mode where applicable.

## Acceptance criteria
- Disabled by default; enable via config (e.g. `MYSQL_MCP_WRITES=1`).
- Audit log entry for every executed write when audit logging is enabled.
EOF
)"

create_issue "Enhancement: richer EXPLAIN as structured output (document existing explain_query)" "$(cat <<'EOF'
## Context
Extended mode already exposes `explain_query`. `run_query` can also run `EXPLAIN`, but structured parsing is inconsistent for agents.

## Proposal
- Ensure `explain_query` is the preferred path and document it in README / tool descriptions.
- Optionally add **MySQL 8+ `EXPLAIN FORMAT=JSON`** (or `TREE`) behind a flag for richer structured fields (`type`, `key`, `rows`, `filtered`, `extra`, etc.).
- Normalize legacy vs JSON explain into a stable JSON schema for MCP clients.

## Note
If the goal is only “dedicated tool,” that exists today as `explain_query` when `MYSQL_MCP_EXTENDED=1`; this issue tracks **depth of structure** and docs.
EOF
)"

create_issue "Feature: kill_query (terminate session / query by connection or thread id)" "$(cat <<'EOF'
## Summary
Operators need to cancel runaway queries when `list_connections` / status tools show pressure.

## Proposal
- Add `kill_query` with `process_id` / `connection_id` (match `INFORMATION_SCHEMA.PROCESSLIST` and server capabilities).
- Guard behind `MYSQL_MCP_KILL_QUERY=1` or admin-only role; refuse without permission.
- Document MySQL vs MariaDB differences (`KILL QUERY` vs `KILL`).

## Safety
- Never kill system threads; validate ID is numeric and owned by permitted user when possible.
EOF
)"

create_issue "Feature: add_connection — register a new named DSN at runtime (no restart)" "$(cat <<'EOF'
## Summary
Adding a new database today requires config/env restart. Dynamic registration improves multi-tenant and ad-hoc workflows.

## Proposal
- New MCP tool `add_connection` (or HTTP `POST /api/connections`) with name, DSN, optional description, SSL flags.
- Persist **only** if explicitly allowed (`MYSQL_MCP_PERSIST_CONNECTIONS` + path); otherwise in-memory for session lifetime.
- Mask secrets in responses; never echo passwords in logs.

## Acceptance criteria
- Integrate with existing `ConnectionManager` and `use_connection`.
- Clear errors on duplicate names or invalid DSN.
EOF
)"

create_issue "Feature: slow_query_log — fetch or tail recent slow query entries" "$(cat <<'EOF'
## Summary
Beyond the `Slow_queries` status counter, operators want **recent statements** without filesystem access on the client.

## Proposal
- Tool `slow_query_log` that, when `slow_query_log=ON` and `log_output` permits, reads from `mysql.slow_log` table (if TABLE) or returns guidance for FILE output.
- Cap rows and redact literal values optionally.

## Constraints
- May be unavailable on managed services; degrade gracefully with clear message.
EOF
)"

# --- Performance & reliability ---

create_issue "Docs / reliability: clarify query timeout env vars and optional server max_execution_time" "$(cat <<'EOF'
## Summary
UAT confusion: “no query timeout” when `max_execution_time=0`. The MCP server **does** support query timeouts via context.

## Existing behavior
- `MYSQL_QUERY_TIMEOUT_SECONDS` (preferred) and `MYSQL_QUERY_TIMEOUT` (milliseconds) configure `QueryTimeout` (default 30s). See `internal/config/config.go`.

## Proposal
- Document prominently in README and MCP setup guides (avoid inventing `MYSQL_MCP_QUERY_TIMEOUT` unless we add an alias).
- Optional stretch: set **session** `max_execution_time` (ms) on pooled connections where MySQL supports it, aligned with `QueryTimeout` — needs careful pool hygiene.

## Acceptance criteria
- Users can find timeout knobs in one place; behavior vs MySQL `max_execution_time` is explained.
EOF
)"

create_issue "Docs / performance: concurrent tool calls and connection pool tuning" "$(cat <<'EOF'
## Summary
Hosts may issue parallel tool calls; default pool may look like “one thread” on the server.

## Existing behavior
- `MYSQL_MAX_OPEN_CONNS` (overrides) and `MYSQL_POOL_SIZE` alias configure max open connections. Defaults: max open 10, idle 5.

## Proposal
- Document recommended `MYSQL_MAX_OPEN_CONNS` for Claude / multi-tool parallelism.
- Optionally surface **active / idle pool stats** in `server_info` or metrics HTTP.

## Acceptance criteria
- Clear guidance for users seeing single-connection behavior under load.
EOF
)"

create_issue "Feature: automatic retry with backoff for transient tool / DB timeouts" "$(cat <<'EOF'
## Summary
Transient timeouts sometimes succeed on retry; today callers must redo manually.

## Proposal
- Configurable `MYSQL_MCP_TOOL_RETRY_MAX` / backoff for idempotent read tools only (e.g. `run_query` with same SQL hash, introspection tools).
- Never retry mutating operations if/when writes exist.

## Acceptance criteria
- Opt-in; default off; logs indicate retry count and reason.
EOF
)"

create_issue "Feature: pagination or streaming for large run_query result sets" "$(cat <<'EOF'
## Summary
`run_query` is capped by `MYSQL_MAX_ROWS` (default 200). Large tables (e.g. 900k+ rows) need keyset pagination or explicit cursor workflow.

## Proposal
- Add `continuation_token` / `offset`+`limit` contract with stable ordering requirement, or document multi-step `run_query` pattern with primary-key range filters.
- Optional: server-side export to temp table / file for admin-only (high complexity).

## Acceptance criteria
- Agents can safely pull large tables in chunks without loading everything into MCP context at once.
EOF
)"

# --- Observability ---

create_issue "Feature: read_audit_log — return recent audit log lines via MCP" "$(cat <<'EOF'
## Summary
`MYSQL_MCP_AUDIT_LOG` writes JSON lines; operators want to read the tail from Claude without shell access.

## Proposal
- Tool `read_audit_log` with `limit`, optional `since` / offset; read-only file access to configured path.
- Enforce max bytes; redact secrets.

## Security
- Disabled by default or require admin capability; path traversal hardening.
EOF
)"

create_issue "Feature: process_list — SHOW PROCESSLIST / PERFORMANCE_SCHEMA threads as a tool" "$(cat <<'EOF'
## Summary
Live diagnostics: see running queries, time, state — distinct from `list_connections` (configured MCP DSNs).

## Proposal
- Tool `process_list` wrapping `SHOW FULL PROCESSLIST` or `performance_schema.threads` where permitted.
- Truncate long `INFO` safely; optional filter by user/host/time.

## Acceptance criteria
- Works on MySQL 8 and MariaDB with documented fallbacks.
EOF
)"

create_issue "Enhancement: expose token usage metrics in server_info or dedicated tool" "$(cat <<'EOF'
## Summary
When `MYSQL_MCP_TOKEN_TRACKING=1`, aggregate metrics exist internally / on HTTP sidecar; exposing them in one call helps agents.

## Proposal
- Extend `server_info` (or add `token_stats`) with rolling totals / last N tool names from `globalTokenMetrics` when tracking enabled.
- Document HTTP `/status` / metrics endpoints vs MCP parity.

## Acceptance criteria
- Single MCP call returns enough signal to tune prompts without scraping logs.
EOF
)"

# --- Security ---

create_issue "Feature: MYSQL_MCP_ALLOWED_DATABASES allowlist for queries" "$(cat <<'EOF'
## Summary
Single DB user may see all schemas; some deployments want to **restrict which schemas** MCP tools may touch.

## Proposal
- Config allowlist; `run_query` and introspection tools reject other databases with clear error.
- Apply to `USE db`, qualified names, and `information_schema` filtering where applicable.

## Acceptance criteria
- Tests for bypass attempts (backticks, casing); clear docs.
EOF
)"

create_issue "Hardening: enforce read-only at session level (SET SESSION TRANSACTION READ ONLY)" "$(cat <<'EOF'
## Summary
SQL classifier is convention-based; a stricter guarantee helps security reviews.

## Proposal
- On connection checkout (or pool new connection), run `SET SESSION TRANSACTION READ ONLY` when `MYSQL_MCP_STRICT_READ_ONLY=1`.
- Verify compatibility with drivers and replicas; document exceptions (some STATUS commands, etc.).

## Acceptance criteria
- Integration test proving DML fails at server layer when enabled (for standard InnoDB).
EOF
)"

create_issue "Feature: MYSQL_MCP_MASK_COLUMNS — redact sensitive columns in tool outputs" "$(cat <<'EOF'
## Summary
Prevent PII / payment fields from reaching the model even if queries are allowed.

## Proposal
- Config: list of `db.table.column` or regex patterns; replace values in `run_query` / describe / sampling tools with `[REDACTED]`.
- Performance: apply only to string-like columns under size cap.

## Acceptance criteria
- Unit tests for nested JSON and aliased column names where feasible.
EOF
)"

# --- DX ---

create_issue "Feature: health snapshot tool (latency + buffer pool + slow queries + threads)" "$(cat <<'EOF'
## Summary
Reduce round-trips: one tool for “is this instance happy?”

## Proposal
- `mysql:health` or extend `ping` with optional `detailed=true` returning latency, `Threads_running`, buffer pool hit ratio (if available), `Slow_queries` delta optional, etc.
- Keep output small and stable JSON.

## Acceptance criteria
- Works on MySQL 8; graceful degradation on MariaDB / limited privilege users.
EOF
)"

create_issue "Feature: schema diff — compare two databases or two schema snapshots" "$(cat <<'EOF'
## Summary
Migration validation: diff tables, columns, indexes between `db_a` and `db_b`.

## Proposal
- Tool `schema_diff` with `source_database`, `target_database`, optional table filter; output added/removed/changed objects.
- Consider using `information_schema` only (read-only).

## Acceptance criteria
- Handles charset/collation and index renames sensibly; large schemas capped with summary counts.
EOF
)"

create_issue "Feature: search_schema — find tables/columns by name pattern across a catalog" "$(cat <<'EOF'
## Summary
Exploration UX: search `information_schema` for `%geoloc%` across many tables.

## Proposal
- Tool `search_schema` with `pattern`, optional `database`, limits; return schema/table/column matches.

## Acceptance criteria
- Safe LIMITs; no full table scans on huge instances beyond IS limits.
EOF
)"

create_issue "Reliability: auto-reconnect stale pooled connections after MySQL restart" "$(cat <<'EOF'
## Summary
After server restart, stale TCP/pool connections may error until MCP process restarts.

## Proposal
- Enable / tune `sql.DB` health checks (`SetConnMaxLifetime` already exists); consider `db.Ping` on checkout or driver `CheckConnLiveness`.
- Retry once on `driver.ErrBadConn` for read tools.

## Acceptance criteria
- Test or integration simulating connection drop; MCP survives without full host restart.
EOF
)"

echo "Done. Created issues on $REPO (or printed with DRY_RUN=1)."
