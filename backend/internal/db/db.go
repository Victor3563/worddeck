package db

import (
	"context"
	"database/sql"
	"fmt"
	"log/slog"
	"time"

	_ "github.com/lib/pq"
)

const (
	pingTimeout       = 3 * time.Second
	retryInitialDelay = 250 * time.Millisecond
	retryMaxDelay     = 2 * time.Second
)

type PoolConfig struct {
	MaxOpenConns    int
	MaxIdleConns    int
	ConnMaxIdleTime time.Duration
	ConnMaxLifetime time.Duration
	ConnectTimeout  time.Duration
}

func Open(ctx context.Context, url string, pc PoolConfig, log *slog.Logger) (*sql.DB, error) {
	pool, err := sql.Open("postgres", url)
	if err != nil {
		return nil, fmt.Errorf("open db: %w", err)
	}
	pool.SetMaxOpenConns(pc.MaxOpenConns)
	pool.SetMaxIdleConns(pc.MaxIdleConns)
	pool.SetConnMaxIdleTime(pc.ConnMaxIdleTime)
	pool.SetConnMaxLifetime(pc.ConnMaxLifetime)

	deadline := time.Now().Add(pc.ConnectTimeout)
	delay := retryInitialDelay
	for attempt := 1; ; attempt++ {
		pingCtx, cancel := context.WithTimeout(ctx, pingTimeout)
		err = pool.PingContext(pingCtx)
		cancel()
		if err == nil {
			return pool, nil
		}
		if time.Now().After(deadline) || ctx.Err() != nil {
			pool.Close()
			return nil, fmt.Errorf("database is unreachable after %d attempts: %w", attempt, err)
		}
		log.Warn("database is not ready yet, retrying", "attempt", attempt, "error", err.Error())
		select {
		case <-ctx.Done():
			pool.Close()
			return nil, ctx.Err()
		case <-time.After(delay):
		}
		delay = min(delay*2, retryMaxDelay)
	}
}
