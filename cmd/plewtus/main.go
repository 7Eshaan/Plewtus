package main

import (
	"context"
	"log/slog"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"plewtus/internal/cache"
	"plewtus/internal/config"
	"plewtus/internal/orchestrator"
	"plewtus/internal/provider/all"
	"plewtus/internal/reliability"
	"plewtus/internal/server"
	"plewtus/internal/usage"
)

func main() {
	cfg := config.Load()

	// Initialize structured logger
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

	var handler slog.Handler
	opts := &slog.HandlerOptions{Level: level}
	if cfg.Env == "production" {
		handler = slog.NewJSONHandler(os.Stdout, opts)
	} else {
		handler = slog.NewTextHandler(os.Stdout, opts)
	}
	logger := slog.New(handler)
	slog.SetDefault(logger)

	// Initialize provider registry and search orchestrator
	providerRegistry := all.NewRegistry(cfg.DefaultTimeout, cfg.UserAgent)
	searchOrchestrator := orchestrator.New(providerRegistry, cfg.DefaultTimeout, cfg.UserAgent, logger)

	// Configure in-memory search response cache if enabled
	if cfg.CacheEnabled {
		searchCache := cache.New(cache.Options{
			TTL:        cfg.CacheTTL,
			MaxEntries: cfg.CacheMaxEntries,
		})
		searchOrchestrator.WithCache(searchCache)
		logger.Info("search cache initialized",
			slog.Duration("ttl", cfg.CacheTTL),
			slog.Int("max_entries", cfg.CacheMaxEntries),
		)
	}

	// Configure provider circuit breakers
	circuitBreakers := reliability.NewManager(reliability.BreakerOptions{
		FailureThreshold: cfg.CircuitBreakerThreshold,
		Cooldown:         cfg.CircuitBreakerCooldown,
	})
	searchOrchestrator.WithBreakers(circuitBreakers)

	// Configure usage manager for authentication, rate limiting, and quotas
	tiers := map[string]usage.Tier{
		"free": {
			Name:            "free",
			RateLimitPerMin: cfg.FreeTierRateLimit,
			DailyQuota:      cfg.FreeTierDailyQuota,
		},
		"pro": {
			Name:            "pro",
			RateLimitPerMin: cfg.ProTierRateLimit,
			DailyQuota:      cfg.ProTierDailyQuota,
		},
	}
	usageMgr := usage.NewManager(usage.Options{
		AuthRequired: cfg.AuthRequired,
		Tiers:        tiers,
		APIKeys:      cfg.APIKeys,
	})

	srv := server.New(cfg, logger, searchOrchestrator).WithUsageManager(usageMgr)

	// Graceful shutdown handling
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	go func() {
		if err := srv.Start(); err != nil {
			logger.Error("server fatal error", slog.Any("error", err))
			os.Exit(1)
		}
	}()

	<-ctx.Done()
	logger.Info("shutting down gracefully...")

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := srv.Shutdown(shutdownCtx); err != nil {
		logger.Error("server shutdown failed", slog.Any("error", err))
		os.Exit(1)
	}

	logger.Info("server exited cleanly")
}
