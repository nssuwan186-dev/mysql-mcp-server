package dbretry

import (
	"context"
	"database/sql/driver"
	"errors"
	"testing"
	"time"
)

func TestDoRetriesDriverBadConn(t *testing.T) {
	ctx := context.Background()
	var attempts int
	err := Do(ctx, nil, Config{MaxRetries: 3, MaxInterval: time.Millisecond}, 0, func() error {
		attempts++
		if attempts == 1 {
			return driver.ErrBadConn
		}
		return nil
	})
	if err != nil {
		t.Fatalf("Do: %v", err)
	}
	if attempts != 2 {
		t.Fatalf("expected 2 attempts, got %d", attempts)
	}
}

func TestDoNoRetryWhenMaxRetriesZero(t *testing.T) {
	ctx := context.Background()
	var attempts int
	err := Do(ctx, nil, Config{MaxRetries: 0, MaxInterval: time.Millisecond}, 0, func() error {
		attempts++
		return driver.ErrBadConn
	})
	if err == nil {
		t.Fatal("expected error")
	}
	if attempts != 1 {
		t.Fatalf("expected 1 attempt, got %d", attempts)
	}
}

func TestDoPermanentError(t *testing.T) {
	ctx := context.Background()
	want := errors.New("syntax")
	var attempts int
	err := Do(ctx, nil, Config{MaxRetries: 3, MaxInterval: time.Millisecond}, 0, func() error {
		attempts++
		return want
	})
	if !errors.Is(err, want) {
		t.Fatalf("got %v want %v", err, want)
	}
	if attempts != 1 {
		t.Fatalf("expected 1 attempt, got %d", attempts)
	}
}
