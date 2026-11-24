package config

import (
	"os"
	"strings"
	"testing"
	"time"
)

func TestConfigDefaults(t *testing.T) {
	cfg := Load()
	if cfg.Port != 8080 {
		t.Errorf("expected default Port 8080, got %d", cfg.Port)
	}
	if cfg.Host != "0.0.0.0" {
		t.Errorf("expected default Host 0.0.0.0, got %s", cfg.Host)
	}
	if cfg.Addr() != "0.0.0.0:8080" {
		t.Errorf("expected Addr 0.0.0.0:8080, got %s", cfg.Addr())
	}
	if len(cfg.DefaultProviders) != 3 {
		t.Errorf("expected 3 default providers, got %d", len(cfg.DefaultProviders))
	}
	if cfg.DefaultTimeout != 6*time.Second {
		t.Errorf("expected 6s default timeout, got %v", cfg.DefaultTimeout)
	}
	if !cfg.CacheEnabled {
		t.Errorf("expected CacheEnabled default true")
	}
	if cfg.CacheTTL != 600*time.Second {
		t.Errorf("expected CacheTTL 600s, got %v", cfg.CacheTTL)
	}
	if cfg.FreeTierRateLimit != 10 {
		t.Errorf("expected FreeTierRateLimit 10, got %d", cfg.FreeTierRateLimit)
	}
	if cfg.FreeTierDailyQuota != 100 {
		t.Errorf("expected FreeTierDailyQuota 100, got %d", cfg.FreeTierDailyQuota)
	}
	if cfg.CircuitBreakerThreshold != 3 {
		t.Errorf("expected CircuitBreakerThreshold 3, got %d", cfg.CircuitBreakerThreshold)
	}
}

func TestConfigEnvOverrides(t *testing.T) {
	os.Setenv("PLEWTUS_PORT", "9090")
	os.Setenv("PLEWTUS_HOST", "127.0.0.1")
	os.Setenv("PLEWTUS_DEFAULT_PROVIDERS", "duckduckgo,bing")
	os.Setenv("PLEWTUS_DEFAULT_TIMEOUT_MS", "5000")
	os.Setenv("PLEWTUS_CACHE_ENABLED", "false")
	os.Setenv("PLEWTUS_AUTH_REQUIRED", "true")
	os.Setenv("PLEWTUS_API_KEYS", "pro-token:pro,free-token:free,simple-token")
	os.Setenv("PLEWTUS_FREE_TIER_RATE_LIMIT", "15")
	os.Setenv("PLEWTUS_FREE_TIER_DAILY_QUOTA", "250")
	os.Setenv("PLEWTUS_CIRCUIT_BREAKER_THRESHOLD", "5")
	os.Setenv("PLEWTUS_CIRCUIT_BREAKER_COOLDOWN_MS", "45000")

	defer func() {
		os.Unsetenv("PLEWTUS_PORT")
		os.Unsetenv("PLEWTUS_HOST")
		os.Unsetenv("PLEWTUS_DEFAULT_PROVIDERS")
		os.Unsetenv("PLEWTUS_DEFAULT_TIMEOUT_MS")
		os.Unsetenv("PLEWTUS_CACHE_ENABLED")
		os.Unsetenv("PLEWTUS_AUTH_REQUIRED")
		os.Unsetenv("PLEWTUS_API_KEYS")
		os.Unsetenv("PLEWTUS_FREE_TIER_RATE_LIMIT")
		os.Unsetenv("PLEWTUS_FREE_TIER_DAILY_QUOTA")
		os.Unsetenv("PLEWTUS_CIRCUIT_BREAKER_THRESHOLD")
		os.Unsetenv("PLEWTUS_CIRCUIT_BREAKER_COOLDOWN_MS")
	}()

	cfg := Load()
	if cfg.Port != 9090 {
		t.Errorf("expected Port 9090, got %d", cfg.Port)
	}
	if cfg.Host != "127.0.0.1" {
		t.Errorf("expected Host 127.0.0.1, got %s", cfg.Host)
	}
	if cfg.Addr() != "127.0.0.1:9090" {
		t.Errorf("expected Addr 127.0.0.1:9090, got %s", cfg.Addr())
	}
	if len(cfg.DefaultProviders) != 2 || cfg.DefaultProviders[0] != "duckduckgo" || cfg.DefaultProviders[1] != "bing" {
		t.Errorf("unexpected DefaultProviders: %v", cfg.DefaultProviders)
	}
	if cfg.DefaultTimeout != 5*time.Second {
		t.Errorf("expected 5s default timeout, got %v", cfg.DefaultTimeout)
	}
	if cfg.CacheEnabled {
		t.Errorf("expected CacheEnabled false")
	}
	if !cfg.AuthRequired {
		t.Errorf("expected AuthRequired true")
	}
	if cfg.APIKeys["pro-token"] != "pro" || cfg.APIKeys["free-token"] != "free" || cfg.APIKeys["simple-token"] != "pro" {
		t.Errorf("unexpected APIKeys map: %v", cfg.APIKeys)
	}
	if cfg.FreeTierRateLimit != 15 {
		t.Errorf("expected FreeTierRateLimit 15, got %d", cfg.FreeTierRateLimit)
	}
	if cfg.FreeTierDailyQuota != 250 {
		t.Errorf("expected FreeTierDailyQuota 250, got %d", cfg.FreeTierDailyQuota)
	}
	if cfg.CircuitBreakerThreshold != 5 {
		t.Errorf("expected CircuitBreakerThreshold 5, got %d", cfg.CircuitBreakerThreshold)
	}
	if cfg.CircuitBreakerCooldown != 45*time.Second {
		t.Errorf("expected CircuitBreakerCooldown 45s, got %v", cfg.CircuitBreakerCooldown)
	}
}

// Unusable values fall back to their defaults and are reported in Warnings;
// valid configuration produces no warnings.
func TestConfigWarnings(t *testing.T) {
	if cfg := Load(); len(cfg.Warnings) != 0 {
		t.Fatalf("default configuration should load without warnings, got %v", cfg.Warnings)
	}

	t.Setenv("PLEWTUS_DEFAULT_TIMEOUT_MS", "six seconds")
	t.Setenv("PLEWTUS_CACHE_MAX_ENTRIES", "0")
	t.Setenv("PLEWTUS_FREE_TIER_DAILY_QUOTA", "-1")
	t.Setenv("PLEWTUS_PORT", "70000")
	t.Setenv("PLEWTUS_CACHE_ENABLED", "enabled")
	t.Setenv("PLEWTUS_LOG_LEVEL", "verbose")
	t.Setenv("PLEWTUS_TRUSTED_PROXIES", "10.0.0.0/8,not-an-ip")
	t.Setenv("PLEWTUS_API_KEYS", "pro_live_agent123:pro,custom-key:gold")

	cfg := Load()
	if cfg.DefaultTimeout != 6*time.Second || cfg.CacheMaxEntries != 1000 || cfg.FreeTierDailyQuota != 100 ||
		cfg.Port != 8080 || !cfg.CacheEnabled {
		t.Fatalf("unusable values must fall back to defaults, got %+v", cfg)
	}
	all := strings.Join(cfg.Warnings, "\n")
	for _, want := range []string{
		"PLEWTUS_DEFAULT_TIMEOUT_MS", "PLEWTUS_CACHE_MAX_ENTRIES", "PLEWTUS_FREE_TIER_DAILY_QUOTA",
		"PLEWTUS_PORT", "PLEWTUS_CACHE_ENABLED", "verbose", "not-an-ip", "SECURITY", "gold",
	} {
		if !strings.Contains(all, want) {
			t.Errorf("expected a warning mentioning %q, got:\n%s", want, all)
		}
	}
	if len(cfg.Warnings) != 9 {
		t.Errorf("expected 9 warnings, got %d:\n%s", len(cfg.Warnings), all)
	}
}

// A zero daily quota (unlimited) and a zero straggler grace are valid.
func TestConfigZeroValuesAllowedWhereMeaningful(t *testing.T) {
	t.Setenv("PLEWTUS_FREE_TIER_DAILY_QUOTA", "0")
	t.Setenv("PLEWTUS_STRAGGLER_GRACE_MS", "0")
	t.Setenv("PLEWTUS_CACHE_TTL_SECONDS", "0")
	cfg := Load()
	if len(cfg.Warnings) != 0 || cfg.FreeTierDailyQuota != 0 || cfg.StragglerGrace != 0 || cfg.CacheTTL != 0 {
		t.Fatalf("zero values should be accepted, got warnings %v", cfg.Warnings)
	}
}
