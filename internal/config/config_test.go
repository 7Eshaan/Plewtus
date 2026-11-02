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
}

func TestConfigEnvOverrides(t *testing.T) {
	os.Setenv("PLEWTUS_PORT", "9090")
	os.Setenv("PLEWTUS_HOST", "127.0.0.1")
	os.Setenv("PLEWTUS_DEFAULT_PROVIDERS", "duckduckgo,bing")
	os.Setenv("PLEWTUS_DEFAULT_TIMEOUT_MS", "5000")
	defer func() {
		os.Unsetenv("PLEWTUS_PORT")
		os.Unsetenv("PLEWTUS_HOST")
		os.Unsetenv("PLEWTUS_DEFAULT_PROVIDERS")
		os.Unsetenv("PLEWTUS_DEFAULT_TIMEOUT_MS")
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
}
