package config

import (
	"os"
	"strconv"
	"strings"
	"time"
)

// Config holds all configuration options for Plewtus.
type Config struct {
	Host                    string
	Port                    int
	Env                     string
	LogLevel                string
	DefaultProviders        []string
	TrustedProxies          []string
	DefaultTimeout          time.Duration
	ExtractTimeout          time.Duration
	StragglerGrace          time.Duration
	MaxLimit                int
	DefaultLimit            int
	UserAgent               string
	Version                 string
	CacheEnabled            bool
	CacheTTL                time.Duration
	CacheMaxEntries         int
	AuthRequired            bool
	APIKeys                 map[string]string
	FreeTierRateLimit       int
	FreeTierDailyQuota      int
	ProTierRateLimit        int
	ProTierDailyQuota       int
	CircuitBreakerThreshold int
	CircuitBreakerCooldown  time.Duration
	// MarginaliaAPIKey authenticates against the Marginalia Search API.
	MarginaliaAPIKey string
	// BraveAPIKey enables Brave's official Search API (empty: scrape the HTML page).
	BraveAPIKey string
	// Commit is the VCS revision the binary was built from, when known.
	Commit string
	// Warnings describe settings that could not be used as given and were
	// replaced by their defaults, plus risky settings such as demo API keys.
	// They are logged at startup.
	Warnings []string
}

// Load loads configuration from environment variables with sane defaults.
// For local development, an optional .env file (path overridable with
// PLEWTUS_ENV_FILE) is read first; already-set variables take precedence.
// Values that cannot be used (not a number, out of range, unknown names) fall
// back to their defaults and are described in Warnings.
func Load() *Config {
	_ = loadDotEnv(getEnv("PLEWTUS_ENV_FILE", ".env"))
	l := &loader{}
	cfg := &Config{
		Host:             getEnv("PLEWTUS_HOST", getEnv("HOST", "0.0.0.0")),
		Port:             l.port(),
		Env:              getEnv("PLEWTUS_ENV", getEnv("ENV", "development")),
		LogLevel:         getEnv("PLEWTUS_LOG_LEVEL", getEnv("LOG_LEVEL", "info")),
		DefaultProviders: getEnvAsSlice("PLEWTUS_DEFAULT_PROVIDERS", []string{"duckduckgo", "bing", "yahoo"}),
		TrustedProxies:   getEnvAsSlice("PLEWTUS_TRUSTED_PROXIES", nil),
		DefaultTimeout:   l.durationMs("PLEWTUS_DEFAULT_TIMEOUT_MS", 6000, 1),
		ExtractTimeout:   l.durationMs("PLEWTUS_EXTRACT_TIMEOUT_MS", 8000, 1),
		StragglerGrace:   l.durationMs("PLEWTUS_STRAGGLER_GRACE_MS", 1500, 0),
		MaxLimit:         l.int("PLEWTUS_MAX_LIMIT", 50, 1),
		DefaultLimit:     l.int("PLEWTUS_DEFAULT_LIMIT", 10, 1),
		UserAgent: getEnv(
			"PLEWTUS_USER_AGENT",
			"Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/128.0.0.0 Safari/537.36",
		),
		Version:                 Version,
		Commit:                  Commit(),
		CacheEnabled:            l.bool("PLEWTUS_CACHE_ENABLED", true),
		CacheTTL:                time.Duration(l.int("PLEWTUS_CACHE_TTL_SECONDS", 600, 0)) * time.Second,
		CacheMaxEntries:         l.int("PLEWTUS_CACHE_MAX_ENTRIES", 1000, 1),
		AuthRequired:            l.bool("PLEWTUS_AUTH_REQUIRED", false),
		APIKeys:                 getEnvAsKeyMap("PLEWTUS_API_KEYS"),
		FreeTierRateLimit:       l.int("PLEWTUS_FREE_TIER_RATE_LIMIT", 10, 1),
		FreeTierDailyQuota:      l.int("PLEWTUS_FREE_TIER_DAILY_QUOTA", 100, 0),
		ProTierRateLimit:        l.int("PLEWTUS_PRO_TIER_RATE_LIMIT", 60, 1),
		ProTierDailyQuota:       l.int("PLEWTUS_PRO_TIER_DAILY_QUOTA", 2000, 0),
		CircuitBreakerThreshold: l.int("PLEWTUS_CIRCUIT_BREAKER_THRESHOLD", 3, 1),
		CircuitBreakerCooldown:  l.durationMs("PLEWTUS_CIRCUIT_BREAKER_COOLDOWN_MS", 30000, 1),
		MarginaliaAPIKey:        getEnv("PLEWTUS_MARGINALIA_API_KEY", "public"),
		BraveAPIKey:             getEnv("PLEWTUS_BRAVE_API_KEY", ""),
	}
	l.checkLogLevel(cfg.LogLevel)
	l.checkTrustedProxies(cfg.TrustedProxies)
	l.checkAPIKeys(cfg.APIKeys)
	cfg.Warnings = l.warnings
	return cfg
}

// Addr returns the formatted address for the HTTP server.
func (c *Config) Addr() string {
	return c.Host + ":" + strconv.Itoa(c.Port)
}

func getEnv(key, defaultVal string) string {
	if val, exists := os.LookupEnv(key); exists && strings.TrimSpace(val) != "" {
		return strings.TrimSpace(val)
	}
	return defaultVal
}

func getEnvAsSlice(key string, defaultVal []string) []string {
	valStr := getEnv(key, "")
	if valStr == "" {
		return defaultVal
	}
	parts := strings.Split(valStr, ",")
	var result []string
	for _, p := range parts {
		trimmed := strings.TrimSpace(p)
		if trimmed != "" {
			result = append(result, trimmed)
		}
	}
	if len(result) == 0 {
		return defaultVal
	}
	return result
}

func getEnvAsKeyMap(key string) map[string]string {
	valStr := getEnv(key, "")
	res := make(map[string]string)
	if valStr == "" {
		return res
	}

	parts := strings.Split(valStr, ",")
	for _, p := range parts {
		trimmed := strings.TrimSpace(p)
		if trimmed == "" {
			continue
		}
		if colon := strings.Index(trimmed, ":"); colon != -1 {
			k := strings.TrimSpace(trimmed[:colon])
			tier := strings.ToLower(strings.TrimSpace(trimmed[colon+1:]))
			if k != "" {
				if tier == "" {
					tier = "pro"
				}
				res[k] = tier
			}
		} else {
			res[trimmed] = "pro"
		}
	}
	return res
}
