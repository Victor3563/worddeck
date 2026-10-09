package app

import (
	"context"
	"log/slog"

	"github.com/worddeck/internal/config"
	"github.com/worddeck/internal/db"
)

const (
	migrateMaxOpenConns = 2
	migrateMaxIdleConns = 1
)

func Migrate(ctx context.Context, cfg config.Config, log *slog.Logger, action string) error {
	pool, err := db.Open(ctx, cfg.DatabaseURL, db.PoolConfig{
		MaxOpenConns:    migrateMaxOpenConns,
		MaxIdleConns:    migrateMaxIdleConns,
		ConnMaxIdleTime: cfg.DBConnMaxIdleTime,
		ConnMaxLifetime: cfg.DBConnMaxLifetime,
		ConnectTimeout:  cfg.DBConnectTimeout,
	}, log)
	if err != nil {
		return err
	}
	defer pool.Close()
	return db.Migrate(ctx, pool, action, log)
}
