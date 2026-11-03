package main

import (
	"context"
	"log/slog"
	"os"
	"os/signal"
	"strings"
	"syscall"

	"plewtus/internal/cache"
	"plewtus/internal/config"
	"plewtus/internal/extractor"
	"plewtus/internal/mcp"
	"plewtus/internal/orchestrator"
	"plewtus/internal/provider/all"
	"plewtus/internal/reliability"
)

func main() {
	cfg := config.Load()

	// Direct all logs to stderr so stdout remains clean for MCP JSON-RPC messages
	var level slog.Level
	switch strings.ToLower(cfg.LogLevel) {
	case "debug":
		level = slog.LevelDebug
	case "warn":
		level = slog.LevelWarn
	case "error":
		level = slog.LevelError
	default:
		level = slog.LevelInfo
	}
	logger := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: level}))

	// Initialize provider registry and orchestrator
	providerRegistry := all.NewRegistry(cfg.DefaultTimeout, cfg.UserAgent)
	ext := extractor.New(extractor.ExtractorOptions{
		Timeout:   cfg.DefaultTimeout,
		UserAgent: cfg.UserAgent,
		Logger:    logger,
	})

	searchOrchestrator := orchestrator.New(providerRegistry, cfg.DefaultTimeout, cfg.UserAgent, logger)
	searchOrchestrator.WithExtractor(ext)

	// Configure in-memory cache if enabled
	if cfg.CacheEnabled {
		searchCache := cache.New(cache.Options{
			TTL:        cfg.CacheTTL,
			MaxEntries: cfg.CacheMaxEntries,
		})
		searchOrchestrator.WithCache(searchCache)
	}

	// Configure circuit breakers
	breakers := reliability.NewManager(reliability.BreakerOptions{
		FailureThreshold: cfg.CircuitBreakerThreshold,
		Cooldown:         cfg.CircuitBreakerCooldown,
	})
	searchOrchestrator.WithBreakers(breakers)

	// Create and start MCP stdio server
	srv := mcp.NewServer(searchOrchestrator, ext, os.Stdin, os.Stdout, logger)

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	if err := srv.Serve(ctx); err != nil && err != context.Canceled {
		logger.Error("mcp server error", slog.Any("error", err))
		os.Exit(1)
	}
}
