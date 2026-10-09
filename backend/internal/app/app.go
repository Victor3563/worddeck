package app

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"log/slog"
	"net"
	"net/http"
	"strconv"

	"github.com/worddeck/internal/auth"
	"github.com/worddeck/internal/config"
	"github.com/worddeck/internal/db"
	"github.com/worddeck/internal/httpapi"
	"github.com/worddeck/internal/service"
	"github.com/worddeck/internal/store"
)

type App struct {
	cfg     config.Config
	log     *slog.Logger
	version string
	pool    *sql.DB
	api     *httpapi.Server
	srv     *http.Server
}

func New(ctx context.Context, cfg config.Config, log *slog.Logger, version string) (*App, error) {
	if err := cfg.ValidateServe(); err != nil {
		return nil, err
	}
	pool, err := db.Open(ctx, cfg.DatabaseURL, db.PoolConfig{
		MaxOpenConns:    cfg.DBMaxOpenConns,
		MaxIdleConns:    cfg.DBMaxIdleConns,
		ConnMaxIdleTime: cfg.DBConnMaxIdleTime,
		ConnMaxLifetime: cfg.DBConnMaxLifetime,
		ConnectTimeout:  cfg.DBConnectTimeout,
	}, log)
	if err != nil {
		return nil, err
	}

	st := store.New(pool)
	accounts, err := service.NewAccounts(st, log, auth.DefaultArgon2, cfg.SessionTTL)
	if err != nil {
		pool.Close()
		return nil, err
	}
	api := httpapi.New(httpapi.Deps{
		Accounts:       accounts,
		Library:        service.NewLibrary(st, log),
		Shares:         service.NewShares(st),
		Study:          service.NewStudy(st),
		DB:             st,
		Log:            log,
		Version:        version,
		MaxImportBytes: cfg.MaxImportBytes,
		SecureCookies:  cfg.CookieSecure,
	})
	srv := &http.Server{
		Addr:              net.JoinHostPort("", strconv.Itoa(cfg.Port)),
		Handler:           api.Handler(),
		ReadHeaderTimeout: cfg.HTTPReadHeaderTimeout,
		ReadTimeout:       cfg.HTTPReadTimeout,
		WriteTimeout:      cfg.HTTPWriteTimeout,
		IdleTimeout:       cfg.HTTPIdleTimeout,
		ErrorLog:          slog.NewLogLogger(log.Handler(), slog.LevelWarn),
	}
	return &App{cfg: cfg, log: log, version: version, pool: pool, api: api, srv: srv}, nil
}

func (a *App) Run(ctx context.Context) error {
	errCh := make(chan error, 1)
	go func() {
		a.log.Info("http server listening", "port", a.cfg.Port, "version", a.version)
		errCh <- a.srv.ListenAndServe()
	}()

	select {
	case err := <-errCh:
		if !errors.Is(err, http.ErrServerClosed) {
			return err
		}
		return nil
	case <-ctx.Done():
	}

	a.log.Info("shutdown signal received, draining", "timeout", a.cfg.ShutdownTimeout.String())
	a.api.SetDraining()
	shutdownCtx, cancel := context.WithTimeout(context.Background(), a.cfg.ShutdownTimeout)
	defer cancel()
	if err := a.srv.Shutdown(shutdownCtx); err != nil {
		return fmt.Errorf("graceful shutdown: %w", err)
	}
	a.log.Info("server stopped cleanly")
	return nil
}

func (a *App) Close() error {
	return a.pool.Close()
}
