package db

import (
	"context"
	"database/sql"
	"embed"
	"fmt"
	"io/fs"
	"log/slog"
	"path"

	"github.com/pressly/goose/v3"
	"github.com/pressly/goose/v3/lock"
)

//go:embed migrations/*.sql
var migrationsFS embed.FS

const (
	MigrateUp     = "up"
	MigrateDown   = "down"
	MigrateStatus = "status"
)

// goose берёт advisory-lock, поэтому migrate можно запускать в нескольких экземплярах одновременно.
func Migrate(ctx context.Context, pool *sql.DB, action string, log *slog.Logger) error {
	fsys, err := fs.Sub(migrationsFS, "migrations")
	if err != nil {
		return err
	}
	locker, err := lock.NewPostgresSessionLocker()
	if err != nil {
		return err
	}
	p, err := goose.NewProvider(goose.DialectPostgres, pool, fsys, goose.WithSessionLocker(locker))
	if err != nil {
		return fmt.Errorf("init migrations: %w", err)
	}

	switch action {
	case MigrateUp:
		res, err := p.Up(ctx)
		for _, r := range res {
			logResult(log, r)
		}
		if err != nil {
			return err
		}
		version, err := p.GetDBVersion(ctx)
		if err != nil {
			return err
		}
		log.Info("migrations complete", "applied", len(res), "version", version)
	case MigrateDown:
		res, err := p.Down(ctx)
		if res != nil {
			logResult(log, res)
		}
		if err != nil {
			return err
		}
	case MigrateStatus:
		list, err := p.Status(ctx)
		if err != nil {
			return err
		}
		for _, st := range list {
			log.Info("migration status", "version", st.Source.Version, "state", string(st.State), "applied_at", st.AppliedAt)
		}
	default:
		return fmt.Errorf("unknown migrate action %q; use %s | %s | %s", action, MigrateUp, MigrateDown, MigrateStatus)
	}
	return nil
}

func logResult(log *slog.Logger, r *goose.MigrationResult) {
	attrs := []any{"version", r.Source.Version, "file", path.Base(r.Source.Path),
		"direction", r.Direction, "duration", r.Duration.String()}
	if r.Error != nil {
		log.Error("migration failed", append(attrs, "error", r.Error.Error())...)
		return
	}
	log.Info("migration applied", attrs...)
}
