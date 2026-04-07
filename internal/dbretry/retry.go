// Package dbretry provides transient-error detection and bounded retries for database/sql.
package dbretry

import (
	"context"
	"database/sql"
	"database/sql/driver"
	"errors"
	"net"
	"time"

	"github.com/cenkalti/backoff/v4"
	"github.com/go-sql-driver/mysql"
)

// Config controls retry behavior for pool operations.
type Config struct {
	MaxRetries  int
	MaxInterval time.Duration
}

// DefaultConfig matches historical internal/mysql Client retry defaults.
func DefaultConfig() Config {
	return Config{
		MaxRetries:  3,
		MaxInterval: 10 * time.Second,
	}
}

// IsTransientError reports whether err is worth retrying (network blip, bad pooled
// connection after server restart, lock contention, etc.). Context cancellation
// and obvious SQL errors are not transient.
func IsTransientError(err error) bool {
	if err == nil {
		return false
	}
	if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
		return false
	}

	var netErr net.Error
	if errors.As(err, &netErr) {
		return true
	}

	if errors.Is(err, mysql.ErrInvalidConn) || errors.Is(err, driver.ErrBadConn) {
		return true
	}

	var mysqlErr *mysql.MySQLError
	if errors.As(err, &mysqlErr) {
		switch mysqlErr.Number {
		case 1040: // ER_CON_COUNT_ERROR
			return true
		case 1213: // ER_LOCK_DEADLOCK
			return true
		case 1205: // ER_LOCK_WAIT_TIMEOUT
			return true
		}
	}

	return false
}

// ShouldWarmPool reports whether a ping may help refresh the pool after this error.
func ShouldWarmPool(err error) bool {
	return errors.Is(err, driver.ErrBadConn) || errors.Is(err, mysql.ErrInvalidConn)
}

// Do runs op until success, a non-transient error, or retries are exhausted.
// If db is non-nil and pingTimeout > 0, a successful Ping is attempted before
// returning a retryable error when the failure looks like a bad pooled connection,
// helping recovery after MySQL restarts (issue #121).
func Do(ctx context.Context, db *sql.DB, cfg Config, pingTimeout time.Duration, op func() error) error {
	if cfg.MaxRetries <= 0 {
		return op()
	}

	ebo := backoff.NewExponentialBackOff()
	ebo.MaxInterval = cfg.MaxInterval
	if ebo.MaxInterval <= 0 {
		ebo.MaxInterval = 10 * time.Second
	}
	ebo.MaxElapsedTime = 0

	bo := backoff.WithContext(ebo, ctx)
	b := backoff.WithMaxRetries(bo, uint64(cfg.MaxRetries))

	return backoff.Retry(func() error {
		err := op()
		if err == nil {
			return nil
		}
		if !IsTransientError(err) {
			return backoff.Permanent(err)
		}
		if db != nil && pingTimeout > 0 && ShouldWarmPool(err) {
			pctx, cancel := context.WithTimeout(ctx, pingTimeout)
			_ = db.PingContext(pctx)
			cancel()
		}
		return err
	}, b)
}
