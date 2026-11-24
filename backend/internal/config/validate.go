package config

import (
	"fmt"
	"net/netip"
	"runtime/debug"
	"strconv"
	"strings"
	"time"
)

// Version is the release version reported by /health and the MCP server.
// Builds can set it with -ldflags "-X plewtus/internal/config.Version=1.2.3".
var Version = "0.1.0"

// commit can be set at build time with
// -ldflags "-X plewtus/internal/config.commit=<sha>".
var commit = ""

// Commit returns the VCS revision the binary was built from, or "" when
// unknown. In order: the -ldflags value; the revision Go embeds when building
// inside a git checkout (12 characters, "-dirty" for a modified tree); then
// PLEWTUS_COMMIT or RENDER_GIT_COMMIT from the environment (Docker builds
// exclude .git, so hosts that know the deployed commit can supply it).
func Commit() string {
	if commit != "" {
		return commit
	}
	if revision := buildRevision(); revision != "" {
		return revision
	}
	for _, key := range []string{"PLEWTUS_COMMIT", "RENDER_GIT_COMMIT"} {
		if v := getEnv(key, ""); v != "" {
			if len(v) > 12 {
				v = v[:12]
			}
			return v
		}
	}
	return ""
}

func buildRevision() string {
	info, ok := debug.ReadBuildInfo()
	if !ok {
		return ""
	}
	var revision string
	var modified bool
	for _, s := range info.Settings {
		switch s.Key {
		case "vcs.revision":
			revision = s.Value
		case "vcs.modified":
			modified = s.Value == "true"
		}
	}
	if len(revision) > 12 {
		revision = revision[:12]
	}
	if revision != "" && modified {
		revision += "-dirty"
	}
	return revision
}

// demoAPIKeys are keys published in this repository's examples and docs.
// Anyone who has read them can use a server that accepts them.
var demoAPIKeys = map[string]bool{
	"pro_live_agent123":              true,
	"free_demo_key":                  true,
	"sk_live_agent123":               true,
	"sk_test_free456":                true,
	"pro-api-key":                    true,
	"replace-with-a-long-random-key": true,
}

// knownTiers are the tiers the usage manager has limits for.
var knownTiers = map[string]bool{"free": true, "pro": true}

// loader reads typed settings and records why a value was not used.
type loader struct {
	warnings []string
}

func (l *loader) warnf(format string, args ...any) {
	l.warnings = append(l.warnings, fmt.Sprintf(format, args...))
}

// int reads an integer of at least min; anything else falls back to def.
func (l *loader) int(key string, def, min int) int {
	raw := getEnv(key, "")
	if raw == "" {
		return def
	}
	val, err := strconv.Atoi(raw)
	if err != nil {
		l.warnf("%s=%q is not a whole number; using the default %d", key, raw, def)
		return def
	}
	if val < min {
		l.warnf("%s=%d is below the minimum %d; using the default %d", key, val, min, def)
		return def
	}
	return val
}

// durationMs reads a millisecond count of at least minMs.
func (l *loader) durationMs(key string, defMs, minMs int) time.Duration {
	return time.Duration(l.int(key, defMs, minMs)) * time.Millisecond
}

// port reads PLEWTUS_PORT, else PORT (as hosting platforms set it), else 8080.
func (l *loader) port() int {
	key := "PLEWTUS_PORT"
	if getEnv(key, "") == "" {
		key = "PORT"
	}
	p := l.int(key, 8080, 1)
	if p > 65535 {
		l.warnf("%s=%d is not a valid port; using the default 8080", key, p)
		return 8080
	}
	return p
}

// bool reads 1/true/yes/on or 0/false/no/off; anything else falls back to def.
func (l *loader) bool(key string, def bool) bool {
	raw := strings.ToLower(getEnv(key, ""))
	switch raw {
	case "":
		return def
	case "1", "true", "yes", "on":
		return true
	case "0", "false", "no", "off":
		return false
	}
	l.warnf("%s=%q is not true or false; using the default %t", key, raw, def)
	return def
}

func (l *loader) checkLogLevel(level string) {
	switch strings.ToLower(strings.TrimSpace(level)) {
	case "debug", "info", "warn", "warning", "error":
	default:
		l.warnf("log level %q is not debug, info, warn or error; using info", level)
	}
}

func (l *loader) checkTrustedProxies(entries []string) {
	for _, raw := range entries {
		if _, err := netip.ParsePrefix(raw); err == nil {
			continue
		}
		if _, err := netip.ParseAddr(raw); err == nil {
			continue
		}
		l.warnf("PLEWTUS_TRUSTED_PROXIES entry %q is not an IP address or CIDR; it is ignored", raw)
	}
}

func (l *loader) checkAPIKeys(keys map[string]string) {
	for key, tier := range keys {
		if demoAPIKeys[key] {
			l.warnf("SECURITY: PLEWTUS_API_KEYS contains %q, an example key published in this repository; anyone can use it. Replace it with a long random key (for example: openssl rand -hex 32)", key)
		}
		if !knownTiers[tier] {
			l.warnf("PLEWTUS_API_KEYS gives a key the unknown tier %q; it gets free-tier limits", tier)
		}
	}
}
