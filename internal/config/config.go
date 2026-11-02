package config

import (
	"os"
	"strconv"
	"strings"
	"time"
)

// Config holds all configuration options for Plewtus.
type Config struct {
	Host             string
	Port             int
	Env              string
	LogLevel         string
	DefaultProviders []string
	DefaultTimeout   time.Duration
	MaxTimeout       time.Duration
	MaxLimit         int
	DefaultLimit     int
	UserAgent        string
	Version          string
}

// Load loads configuration from environment variables with sane defaults.
func Load() *Config {
	return &Config{
		Host:             getEnv("PLEWTUS_HOST", getEnv("HOST", "0.0.0.0")),
		Port:             getEnvAsInt("PLEWTUS_PORT", getEnvAsInt("PORT", 8080)),
		Env:              getEnv("PLEWTUS_ENV", getEnv("ENV", "development")),
		LogLevel:         getEnv("PLEWTUS_LOG_LEVEL", getEnv("LOG_LEVEL", "info")),
		DefaultProviders: getEnvAsSlice("PLEWTUS_DEFAULT_PROVIDERS", []string{"duckduckgo", "bing", "yahoo"}),
		DefaultTimeout:   getEnvAsDurationMs("PLEWTUS_DEFAULT_TIMEOUT_MS", 3000),
		MaxTimeout:       getEnvAsDurationMs("PLEWTUS_MAX_TIMEOUT_MS", 10000),
		MaxLimit:         getEnvAsInt("PLEWTUS_MAX_LIMIT", 50),
		DefaultLimit:     getEnvAsInt("PLEWTUS_DEFAULT_LIMIT", 10),
		UserAgent: getEnv(
			"PLEWTUS_USER_AGENT",
			"Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/128.0.0.0 Safari/537.36",
		),
		Version: "0.1.0",
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
