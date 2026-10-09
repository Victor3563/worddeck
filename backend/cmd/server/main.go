package main

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"syscall"
	_ "time/tzdata"

	"github.com/worddeck/internal/app"
	"github.com/worddeck/internal/config"
	"github.com/worddeck/internal/db"
	"github.com/worddeck/internal/logger"
)

var version = "dev"

const serviceName = "worddeck-api"

const (
	exitOK     = 0
	exitError  = 1
	exitConfig = 2
)

const usage = "serve | migrate [up|down|status]"

type command func(ctx context.Context, cfg config.Config, log *slog.Logger, args []string) error

var commands = map[string]command{
	"serve":   serve,
	"migrate": migrate,
}

func main() {
	os.Exit(run(os.Args[1:]))
}

func run(args []string) int {
	cmd := "serve"
	if len(args) > 0 {
		cmd, args = args[0], args[1:]
	}
	runCmd, ok := commands[cmd]
	if !ok {
		fmt.Fprintf(os.Stderr, "unknown command %q; use %s\n", cmd, usage)
		return exitConfig
	}

	cfg, err := config.Load()
	log := logger.New(os.Stdout, cfg.LogLevel, cfg.LogFormat, serviceName)
	if err != nil {
		log.Error("invalid configuration", "error", err.Error())
		return exitConfig
	}

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()
	go func() {
		<-ctx.Done()
		stop()
	}()

	if err := runCmd(ctx, cfg, log, args); err != nil {
		log.Error("exiting with error", "command", cmd, "error", err.Error())
		return exitError
	}
	return exitOK
}

func serve(ctx context.Context, cfg config.Config, log *slog.Logger, _ []string) error {
	a, err := app.New(ctx, cfg, log, version)
	if err != nil {
		return err
	}
	defer func() {
		if err := a.Close(); err != nil {
			log.Warn("closing application", "error", err.Error())
		}
	}()
	return a.Run(ctx)
}

func migrate(ctx context.Context, cfg config.Config, log *slog.Logger, args []string) error {
	action := db.MigrateUp
	if len(args) > 0 {
		action = args[0]
	}
	return app.Migrate(ctx, cfg, log, action)
}
