// Package app wires configuration into the search engine components shared by
// the HTTP server and the MCP stdio server.
package app

import (
	"log/slog"
	"os"
	"strings"
	"time"

	"plewtus/internal/cache"
	"plewtus/internal/config"
	"plewtus/internal/extractor"
	"plewtus/internal/orchestrator"
	"plewtus/internal/provider/all"
	"plewtus/internal/reliability"
)

// maxPageFetchTimeout caps how long one page fetch may take during extraction.
const maxPageFetchTimeout = 5 * time.Second

// Components are the long-lived engine services built from configuration.
type Components struct {
	Orchestrator *orchestrator.Orchestrator
	Extractor    *extractor.Extractor
}

// Build constructs the provider registry, extractor, cache, circuit breakers,
// and orchestrator from configuration.
func Build(cfg *config.Config, logger *slog.Logger) *Components {
	for _, warning := range cfg.Warnings {
		logger.Warn("configuration: " + warning)
	}
	registry := all.NewRegistry(cfg.DefaultTimeout, cfg.UserAgent, all.Options{
		MarginaliaAPIKey: cfg.MarginaliaAPIKey,
		BraveAPIKey:      cfg.BraveAPIKey,
	})
	for _, name := range cfg.DefaultProviders {
		if _, ok := registry.Get(strings.ToLower(name)); !ok {
			logger.Error("configuration: PLEWTUS_DEFAULT_PROVIDERS names an unknown engine; searches that use the defaults will be rejected",
				slog.String("engine", name), slog.Any("available", registry.Names()))
		}
	}
	// A single page gets less than the whole extraction stage, so one slow
	// site cannot hold back the pages that already finished.
	pageTimeout := cfg.ExtractTimeout
	if pageTimeout > maxPageFetchTimeout {
		pageTimeout = maxPageFetchTimeout
	}
	ext := extractor.New(extractor.ExtractorOptions{
		Timeout:   pageTimeout,
		UserAgent: cfg.UserAgent,
		Logger:    logger,
	})

	orch := orchestrator.New(registry, cfg.DefaultTimeout, cfg.UserAgent, logger).
		WithExtractor(ext).
		WithExtractTimeout(cfg.ExtractTimeout).
		WithStragglerGrace(cfg.StragglerGrace).
		WithFallbackProviders(FallbackProviders(cfg)).
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

	if extractor.PDFSupported() {
		logger.Debug("pdf extraction available (pdftotext found)")
	} else {
		logger.Warn("pdf extraction unavailable: pdftotext (poppler-utils) is not installed; PDF results will report content_error")
	}

	return &Components{Orchestrator: orch, Extractor: ext}
}

// FallbackProviders lists the engines that may stand in when every requested
// engine's circuit is open, most dependable first: the default engines, then
// Marginalia (official API), then Brave only when it has an API key (its HTML
// page answers non-browser clients with a captcha). Qwant is left out because
// it only serves some countries.
func FallbackProviders(cfg *config.Config) []string {
	names := append([]string(nil), cfg.DefaultProviders...)
	names = append(names, "marginalia")
	if strings.TrimSpace(cfg.BraveAPIKey) != "" {
		names = append(names, "brave")
	}
	return names
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
