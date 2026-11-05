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
}

// Load loads configuration from environment variables with sane defaults.
func Load() *Config {
	return &Config{
		Host:             getEnv("PLEWTUS_HOST", getEnv("HOST", "0.0.0.0")),
		Port:             getEnvAsInt("PLEWTUS_PORT", getEnvAsInt("PORT", 8080)),
		Env:              getEnv("PLEWTUS_ENV", getEnv("ENV", "development")),
		LogLevel:         getEnv("PLEWTUS_LOG_LEVEL", getEnv("LOG_LEVEL", "info")),
		DefaultProviders: getEnvAsSlice("PLEWTUS_DEFAULT_PROVIDERS", []string{"duckduckgo", "bing", "yahoo"}),
		TrustedProxies:   getEnvAsSlice("PLEWTUS_TRUSTED_PROXIES", nil),
		DefaultTimeout:   getEnvAsDurationMs("PLEWTUS_DEFAULT_TIMEOUT_MS", 6000),
		MaxLimit:         getEnvAsInt("PLEWTUS_MAX_LIMIT", 50),
		DefaultLimit:     getEnvAsInt("PLEWTUS_DEFAULT_LIMIT", 50),
		UserAgent: getEnv(
			"PLEWTUS_USER_AGENT",
			"Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/128.0.0.0 Safari/537.36",
		),
		Version:                 "0.1.0",
		CacheEnabled:            getEnvAsBool("PLEWTUS_CACHE_ENABLED", true),
		CacheTTL:                time.Duration(getEnvAsInt("PLEWTUS_CACHE_TTL_SECONDS", 600)) * time.Second,
		CacheMaxEntries:         getEnvAsInt("PLEWTUS_CACHE_MAX_ENTRIES", 1000),
		AuthRequired:            getEnvAsBool("PLEWTUS_AUTH_REQUIRED", false),
		APIKeys:                 getEnvAsKeyMap("PLEWTUS_API_KEYS"),
		FreeTierRateLimit:       getEnvAsInt("PLEWTUS_FREE_TIER_RATE_LIMIT", 10),
		FreeTierDailyQuota:      getEnvAsInt("PLEWTUS_FREE_TIER_DAILY_QUOTA", 100),
		ProTierRateLimit:        getEnvAsInt("PLEWTUS_PRO_TIER_RATE_LIMIT", 60),
		ProTierDailyQuota:       getEnvAsInt("PLEWTUS_PRO_TIER_DAILY_QUOTA", 2000),
		CircuitBreakerThreshold: getEnvAsInt("PLEWTUS_CIRCUIT_BREAKER_THRESHOLD", 3),
		CircuitBreakerCooldown:  getEnvAsDurationMs("PLEWTUS_CIRCUIT_BREAKER_COOLDOWN_MS", 30000),
	}
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

func getEnvAsInt(key string, defaultVal int) int {
	valStr := getEnv(key, "")
	if valStr == "" {
		return defaultVal
	}
	val, err := strconv.Atoi(valStr)
	if err != nil {
		return defaultVal
	}
	return val
}

func getEnvAsBool(key string, defaultVal bool) bool {
	valStr := strings.ToLower(getEnv(key, ""))
	if valStr == "" {
		return defaultVal
	}
	return valStr == "1" || valStr == "true" || valStr == "yes" || valStr == "on"
}

func getEnvAsDurationMs(key string, defaultMs int) time.Duration {
	ms := getEnvAsInt(key, defaultMs)
	return time.Duration(ms) * time.Millisecond
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
