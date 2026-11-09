// Command plewtus runs the Plewtus HTTP search API, or the MCP stdio server
// when invoked as `plewtus mcp`.
package main

import (
	"context"
	"log/slog"
	"os"
	"os/signal"
	"syscall"
	"time"

	"plewtus/internal/app"
	"plewtus/internal/config"
	"plewtus/internal/mcp"
	"plewtus/internal/server"
)

func main() {
	cfg := config.Load()

	if len(os.Args) > 1 {
		switch os.Args[1] {
		case "mcp":
			os.Exit(mcp.Run(cfg))
		case "serve":
			// Explicit form of the default command.
		default:
			slog.Error("unknown command; use `plewtus [serve|mcp]`", slog.String("command", os.Args[1]))
			os.Exit(2)
		}
	}

	opts := &slog.HandlerOptions{Level: app.LogLevel(cfg.LogLevel)}
	var handler slog.Handler = slog.NewTextHandler(os.Stdout, opts)
	if cfg.Env == "production" {
		handler = slog.NewJSONHandler(os.Stdout, opts)
	}
	logger := slog.New(handler)
	slog.SetDefault(logger)

	components := app.Build(cfg, logger)
	srv := server.New(cfg, logger, components.Orchestrator)

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	errCh := make(chan error, 1)
	go func() { errCh <- srv.Start() }()

	select {
	case err := <-errCh:
		if err != nil {
			logger.Error("server fatal error", slog.Any("error", err))
			os.Exit(1)
		}
		return
	case <-ctx.Done():
	}
	logger.Info("shutting down gracefully...")

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := srv.Shutdown(shutdownCtx); err != nil {
		logger.Error("server shutdown failed", slog.Any("error", err))
		os.Exit(1)
	}
	logger.Info("server exited cleanly")
}
