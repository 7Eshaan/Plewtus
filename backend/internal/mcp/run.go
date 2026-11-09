package mcp

import (
	"context"
	"errors"
	"log/slog"
	"os"
	"os/signal"
	"syscall"

	"plewtus/internal/app"
	"plewtus/internal/config"
)

// Run starts the MCP stdio server with production wiring and returns a
// process exit code. Logs go to stderr; stdout carries only JSON-RPC.
func Run(cfg *config.Config) int {
	logger := app.StderrLogger(app.LogLevel(cfg.LogLevel))
	components := app.Build(cfg, logger)

	srv := NewServer(components.Orchestrator, components.Extractor, os.Stdin, os.Stdout, logger, cfg.DefaultProviders).
		WithVersion(cfg.Version).
		WithLimits(cfg.DefaultLimit, cfg.MaxLimit).
		WithProviderNames(components.Orchestrator.ProviderNames())

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	if err := srv.Serve(ctx); err != nil && !errors.Is(err, context.Canceled) {
		logger.Error("mcp server error", slog.Any("error", err))
		return 1
	}
	return 0
}
