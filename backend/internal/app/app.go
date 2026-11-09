// Package app wires configuration into the search engine components shared by
// the HTTP server and the MCP stdio server.
package app

import (
	"log/slog"
	"os"
	"strings"

	"plewtus/internal/cache"
	"plewtus/internal/config"
	"plewtus/internal/extractor"
	"plewtus/internal/orchestrator"
	"plewtus/internal/provider/all"
	"plewtus/internal/reliability"
)

// Components are the long-lived engine services built from configuration.
type Components struct {
	Orchestrator *orchestrator.Orchestrator
	Extractor    *extractor.Extractor
}

// Build constructs the provider registry, extractor, cache, circuit breakers,
// and orchestrator from configuration.
func Build(cfg *config.Config, logger *slog.Logger) *Components {
	registry := all.NewRegistry(cfg.DefaultTimeout, cfg.UserAgent)
	ext := extractor.New(extractor.ExtractorOptions{
		Timeout:   cfg.ExtractTimeout,
		UserAgent: cfg.UserAgent,
		Logger:    logger,
	})

	orch := orchestrator.New(registry, cfg.DefaultTimeout, cfg.UserAgent, logger).
		WithExtractor(ext).
		WithExtractTimeout(cfg.ExtractTimeout).
		WithBreakers(reliability.NewManager(reliability.BreakerOptions{
			FailureThreshold: cfg.CircuitBreakerThreshold,
			Cooldown:         cfg.CircuitBreakerCooldown,
		}))

	if cfg.CacheEnabled {
		orch.WithCache(cache.New(cache.Options{
			TTL:        cfg.CacheTTL,
			MaxEntries: cfg.CacheMaxEntries,
		}))
		logger.Debug("search cache initialized",
			slog.Duration("ttl", cfg.CacheTTL),
			slog.Int("max_entries", cfg.CacheMaxEntries),
		)
	}

	return &Components{Orchestrator: orch, Extractor: ext}
}

// LogLevel parses a textual log level, defaulting to info.
func LogLevel(raw string) slog.Level {
	switch strings.ToLower(strings.TrimSpace(raw)) {
	case "debug":
		return slog.LevelDebug
	case "warn", "warning":
		return slog.LevelWarn
	case "error":
		return slog.LevelError
	default:
		return slog.LevelInfo
	}
}

// StderrLogger returns a text logger on stderr, keeping stdout free for
// MCP JSON-RPC traffic.
func StderrLogger(level slog.Level) *slog.Logger {
	return slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: level}))
}
