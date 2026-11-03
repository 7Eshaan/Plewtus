package config

import (
	"os"
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
	if cfg.DefaultTimeout != 3*time.Second {
		t.Errorf("expected 3s default timeout, got %v", cfg.DefaultTimeout)
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
