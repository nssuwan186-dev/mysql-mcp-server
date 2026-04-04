// internal/mysql/client.go
package mysql

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"net"
	"time"

	"github.com/cenkalti/backoff/v4"
	"github.com/go-sql-driver/mysql"

	"github.com/askdba/mysql-mcp-server/internal/util"
)

type Client struct {
	db           *sql.DB
	maxRows      int
	queryTimeout time.Duration
	retryCfg     RetryConfig
}

type RetryConfig struct {
	MaxRetries int
	MaxBackoff time.Duration
}

type Config struct {
	DSN             string
	MaxRows         int
	QueryTimeoutS   int
	MaxOpenConns    int
	MaxIdleConns    int
	ConnMaxLifetime time.Duration
	ConnMaxIdleTime time.Duration
	PingTimeout     time.Duration
	Retry           RetryConfig
}

func New(cfg Config) (*Client, error) {
	db, err := sql.Open("mysql", cfg.DSN)
	if err != nil {
		return nil, err
	}

	// Apply pool settings with sensible defaults
	maxOpen := cfg.MaxOpenConns
	if maxOpen <= 0 {
		maxOpen = 10
	}
	maxIdle := cfg.MaxIdleConns
	if maxIdle <= 0 {
		maxIdle = 5
	}
	lifetime := cfg.ConnMaxLifetime
	if lifetime <= 0 {
		lifetime = 30 * time.Minute
	}
	idleTime := cfg.ConnMaxIdleTime
	if idleTime <= 0 {
		idleTime = 5 * time.Minute
	}
	pingTimeout := cfg.PingTimeout
	if pingTimeout <= 0 {
		pingTimeout = 5 * time.Second
	}

	db.SetMaxOpenConns(maxOpen)
	db.SetMaxIdleConns(maxIdle)
	db.SetConnMaxLifetime(lifetime)
	db.SetConnMaxIdleTime(idleTime)

	ctx, cancel := context.WithTimeout(context.Background(), pingTimeout)
	defer cancel()
	if err := db.PingContext(ctx); err != nil {
		db.Close()
		return nil, err
	}

	if cfg.Retry.MaxRetries <= 0 {
		cfg.Retry.MaxRetries = 3
	}
	if cfg.Retry.MaxBackoff <= 0 {
		cfg.Retry.MaxBackoff = 10 * time.Second
	}

	return &Client{
		db:           db,
		maxRows:      cfg.MaxRows,
		queryTimeout: time.Duration(cfg.QueryTimeoutS) * time.Second,
		retryCfg:     cfg.Retry,
	}, nil
}

// NewWithDB constructs a Client from an existing *sql.DB.
// This is mainly useful for tests where we use a sqlmock.DB.
func NewWithDB(db *sql.DB, cfg Config) (*Client, error) {
	if db == nil {
		return nil, fmt.Errorf("db is nil")
	}

	if cfg.Retry.MaxRetries <= 0 {
		cfg.Retry.MaxRetries = 3
	}
	if cfg.Retry.MaxBackoff <= 0 {
		cfg.Retry.MaxBackoff = 10 * time.Second
	}

	return &Client{
		db:           db,
		maxRows:      cfg.MaxRows,
		queryTimeout: time.Duration(cfg.QueryTimeoutS) * time.Second,
		retryCfg:     cfg.Retry,
	}, nil
}

func (c *Client) Close() error {
	return c.db.Close()
}

func (c *Client) withTimeout(ctx context.Context) (context.Context, context.CancelFunc) {
	return context.WithTimeout(ctx, c.queryTimeout)
}

func (c *Client) execWithRetry(ctx context.Context, op func(context.Context) error) error {
	bo := backoff.NewExponentialBackOff()
	bo.MaxElapsedTime = c.retryCfg.MaxBackoff
	// We'll also cap the number of retries if MaxRetries > 0
	b := backoff.WithMaxRetries(bo, uint64(c.retryCfg.MaxRetries))

	return backoff.Retry(func() error {
		err := op(ctx)
		if err == nil {
			return nil
		}

		if isTransientError(err) {
			return err // backoff.Retry will call this again
		}

		return backoff.Permanent(err)
	}, b)
}

func isTransientError(err error) bool {
	if err == nil {
		return false
	}

	// Check for network errors
	var netErr net.Error
	if errors.As(err, &netErr) {
		return true
	}

	// Check for MySQL specific errors
	var mysqlErr *mysql.MySQLError
	if errors.As(err, &mysqlErr) {
		switch mysqlErr.Number {
		case 1040: // ER_CON_COUNT_ERROR
			return true
		case 1213: // ER_LOCK_DEADLOCK
			return true
		case 1205: // ER_LOCK_WAIT_TIMEOUT
			return true
		case 2002: // CR_CONNECTION_ERROR
			return true
		case 2003: // CR_CONN_HOST_ERROR
			return true
		case 2006: // CR_SERVER_GONE_ERROR
			return true
		case 2013: // CR_SERVER_LOST
			return true
		}
	}

	// Check for driver-specific errors that are not *mysql.MySQLError
	if errors.Is(err, mysql.ErrInvalidConn) {
		return true
	}

	return false
}

func (c *Client) ListDatabases(ctx context.Context) ([]string, error) {
	var dbs []string
	err := c.execWithRetry(ctx, func(ctx context.Context) error {
		ctx, cancel := c.withTimeout(ctx)
		defer cancel()

		rows, err := c.db.QueryContext(ctx, "SHOW DATABASES")
		if err != nil {
			return err
		}
		defer rows.Close()

		dbs = nil // Reset for retry
		for rows.Next() {
			var name string
			if err := rows.Scan(&name); err != nil {
				return err
			}
			dbs = append(dbs, name)
		}
		return rows.Err()
	})
	return dbs, err
}

func (c *Client) ListTables(ctx context.Context, database string) ([]string, error) {
	if database == "" {
		return nil, fmt.Errorf("database is required")
	}

	dbName, err := util.QuoteIdent(database)
	if err != nil {
		return nil, fmt.Errorf("invalid database name: %w", err)
	}

	var tables []string
	err = c.execWithRetry(ctx, func(ctx context.Context) error {
		ctx, cancel := c.withTimeout(ctx)
		defer cancel()

		if _, err := c.db.ExecContext(ctx, "USE "+dbName); err != nil {
			return err
		}

		rows, err := c.db.QueryContext(ctx, "SHOW TABLES")
		if err != nil {
			return err
		}
		defer rows.Close()

		tables = nil // Reset for retry
		for rows.Next() {
			var name string
			if err := rows.Scan(&name); err != nil {
				return err
			}
			tables = append(tables, name)
		}
		return rows.Err()
	})
	return tables, err
}

func (c *Client) DescribeTable(ctx context.Context, database, table string) ([]map[string]any, error) {
	if database == "" || table == "" {
		return nil, fmt.Errorf("database and table are required")
	}

	dbName, err := util.QuoteIdent(database)
	if err != nil {
		return nil, fmt.Errorf("invalid database name: %w", err)
	}
	tableName, err := util.QuoteIdent(table)
	if err != nil {
		return nil, fmt.Errorf("invalid table name: %w", err)
	}

	var out []map[string]any
	err = c.execWithRetry(ctx, func(ctx context.Context) error {
		ctx, cancel := c.withTimeout(ctx)
		defer cancel()

		if _, err := c.db.ExecContext(ctx, "USE "+dbName); err != nil {
			return err
		}

		rows, err := c.db.QueryContext(ctx, "DESCRIBE "+tableName)
		if err != nil {
			return err
		}
		defer rows.Close()

		cols, err := rows.Columns()
		if err != nil {
			return err
		}

		out = nil // Reset for retry
		for rows.Next() {
			values := make([]any, len(cols))
			ptrs := make([]any, len(cols))
			for i := range values {
				ptrs[i] = &values[i]
			}
			if err := rows.Scan(ptrs...); err != nil {
				return err
			}

			row := map[string]any{}
			for i, col := range cols {
				row[col] = values[i]
			}
			out = append(out, row)
		}
		return rows.Err()
	})
	return out, err
}

// RunQuery is intentionally “read oriented” – callers should enforce SELECT only.
func (c *Client) RunQuery(ctx context.Context, sqlText string, maxRows int) ([]map[string]any, error) {
	if sqlText == "" {
		return nil, fmt.Errorf("sql is required")
	}
	if maxRows <= 0 || maxRows > c.maxRows {
		maxRows = c.maxRows
	}

	var result []map[string]any
	err := c.execWithRetry(ctx, func(ctx context.Context) error {
		ctx, cancel := c.withTimeout(ctx)
		defer cancel()

		rows, err := c.db.QueryContext(ctx, sqlText)
		if err != nil {
			return err
		}
		defer rows.Close()

		cols, err := rows.Columns()
		if err != nil {
			return err
		}

		result = nil // Reset for retry
		count := 0
		for rows.Next() {
			if count >= maxRows {
				break
			}
			values := make([]any, len(cols))
			ptrs := make([]any, len(cols))
			for i := range values {
				ptrs[i] = &values[i]
			}
			if err := rows.Scan(ptrs...); err != nil {
				return err
			}
			row := map[string]any{}
			for i, col := range cols {
				row[col] = values[i]
			}
			result = append(result, row)
			count++
		}
		return rows.Err()
	})
	return result, err
}
