package usage

import (
	"errors"
	"fmt"
	"math"
	"net"
	"net/http"
	"strings"
	"sync"
	"time"
)

var (
	ErrAuthRequired      = errors.New("authentication required: missing API key")
	ErrInvalidAPIKey     = errors.New("invalid or unrecognized API key")
	ErrRateLimitExceeded = errors.New("rate limit exceeded")
	ErrQuotaExceeded     = errors.New("daily request quota exceeded")
)

// Tier defines usage limits for an access tier.
type Tier struct {
	Name            string // "free", "pro", etc.
	RateLimitPerMin int    // Allowed requests per minute
	DailyQuota      int    // Allowed requests per UTC day (0 = unlimited)
}

// Identity represents the authenticated caller or anonymous free client.
type Identity struct {
	Key     string // Unique lookup key: "key:<raw>" or "ip:<addr>"
	Type    string // "api_key" or "ip"
	Tier    string // "free", "pro", etc.
	RawKey  string // Raw API key if present
	Address string // Remote IP address
}

// CheckResult contains rate limit and quota status to attach to HTTP headers.
type CheckResult struct {
	Identity            Identity
	RateLimitLimit      int
	RateLimitRemaining  int
	RateLimitResetSec   int
	QuotaLimit          int
	QuotaRemaining      int
	QuotaResetSec       int
	RetryAfterSec       int
}

// Options configures the UsageManager.
type Options struct {
	AuthRequired bool            // If true, requests without a valid API key are rejected (401)
	Tiers        map[string]Tier // Limits per tier
	APIKeys      map[string]string // map[apiKey]tierName
}

// tokenBucket tracks token bucket rate limiting for a single client.
type tokenBucket struct {
	tokens     float64
	capacity   float64
	fillRate   float64 // tokens per second
	lastRefill time.Time
}

// quotaRecord tracks daily usage for a single client.
type quotaRecord struct {
	day   string
	count int
}

// Stats holds aggregate usage metrics.
type Stats struct {
	TotalRequests       int64 `json:"total_requests"`
	AllowedRequests     int64 `json:"allowed_requests"`
	RateLimitedRequests int64 `json:"rate_limited_requests"`
	QuotaExceededCount  int64 `json:"quota_exceeded_requests"`
	UnauthorizedCount   int64 `json:"unauthorized_requests"`
	ActiveClients       int   `json:"active_clients"`
}

// Manager coordinates API key authentication, rate limiting, and quotas.
type Manager struct {
	mu           sync.Mutex
	authRequired bool
	tiers        map[string]Tier
	apiKeys      map[string]string // apiKey -> tierName
	buckets      map[string]*tokenBucket
	quotas       map[string]*quotaRecord
	stats        Stats
}

// NewManager creates a new usage manager.
func NewManager(opts Options) *Manager {
	tiers := make(map[string]Tier)
	if len(opts.Tiers) == 0 {
		// Sane defaults
		tiers["free"] = Tier{
			Name:            "free",
			RateLimitPerMin: 10,
			DailyQuota:      100,
		}
		tiers["pro"] = Tier{
			Name:            "pro",
			RateLimitPerMin: 60,
			DailyQuota:      2000,
		}
	} else {
		for k, v := range opts.Tiers {
			tiers[k] = v
		}
	}

	apiKeys := make(map[string]string)
	for k, v := range opts.APIKeys {
		apiKeys[k] = v
	}

	return &Manager{
		authRequired: opts.AuthRequired,
		tiers:        tiers,
		apiKeys:      apiKeys,
		buckets:      make(map[string]*tokenBucket),
		quotas:       make(map[string]*quotaRecord),
	}
}

// RegisterKey adds or updates an API key with an assigned tier.
func (m *Manager) RegisterKey(key, tierName string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.apiKeys[key] = tierName
}

// Check inspects the incoming request, authenticates, and validates rate limit and quota.
func (m *Manager) Check(r *http.Request) (*CheckResult, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	now := time.Now().UTC()
	m.stats.TotalRequests++

	// 1. Resolve Identity
	rawKey := extractAPIKey(r)
	var ident Identity

	if rawKey != "" {
		tierName, ok := m.apiKeys[rawKey]
		if !ok {
			m.stats.UnauthorizedCount++
			return nil, ErrInvalidAPIKey
		}
		ident = Identity{
			Key:     "key:" + rawKey,
			Type:    "api_key",
			Tier:    tierName,
			RawKey:  rawKey,
			Address: extractClientIP(r),
		}
	} else {
		if m.authRequired {
			m.stats.UnauthorizedCount++
			return nil, ErrAuthRequired
		}
		ip := extractClientIP(r)
		ident = Identity{
			Key:     "ip:" + ip,
			Type:    "ip",
			Tier:    "free",
			Address: ip,
		}
	}

	tier, exists := m.tiers[ident.Tier]
	if !exists {
		// Fallback to free tier limits if tier configuration is missing
		tier = m.tiers["free"]
	}

	// 2. Daily Quota Check
	today := now.Format("2006-01-02")
	secondsUntilMidnight := int(now.Truncate(24*time.Hour).Add(24*time.Hour).Sub(now).Seconds())

	qRecord, qExists := m.quotas[ident.Key]
	if !qExists || qRecord.day != today {
		qRecord = &quotaRecord{
			day:   today,
			count: 0,
		}
		m.quotas[ident.Key] = qRecord
	}

	quotaLimit := tier.DailyQuota
	quotaRemaining := quotaLimit - qRecord.count
	if quotaLimit > 0 && qRecord.count >= quotaLimit {
		m.stats.QuotaExceededCount++
		res := &CheckResult{
			Identity:          ident,
			RateLimitLimit:    tier.RateLimitPerMin,
			QuotaLimit:        quotaLimit,
			QuotaRemaining:    0,
			QuotaResetSec:     secondsUntilMidnight,
			RetryAfterSec:     secondsUntilMidnight,
		}
		return res, fmt.Errorf("%w for tier %q (limit: %d/day, resets in %ds)",
			ErrQuotaExceeded, tier.Name, quotaLimit, secondsUntilMidnight)
	}

	// 3. Token Bucket Rate Limit Check
	bucket, bExists := m.buckets[ident.Key]
	limitPerMin := tier.RateLimitPerMin
	if limitPerMin <= 0 {
		limitPerMin = 10
	}
	capacity := float64(limitPerMin)
	fillRate := capacity / 60.0 // tokens per second

	if !bExists {
		bucket = &tokenBucket{
			tokens:     capacity,
			capacity:   capacity,
			fillRate:   fillRate,
			lastRefill: now,
		}
		m.buckets[ident.Key] = bucket
	} else {
		// Refill bucket based on elapsed time
		elapsed := now.Sub(bucket.lastRefill).Seconds()
		bucket.tokens = math.Min(bucket.capacity, bucket.tokens+elapsed*bucket.fillRate)
		bucket.lastRefill = now
	}

	if bucket.tokens < 1.0 {
		m.stats.RateLimitedRequests++
		missing := 1.0 - bucket.tokens
		retryAfter := int(math.Ceil(missing / bucket.fillRate))
		if retryAfter <= 0 {
			retryAfter = 1
		}

		res := &CheckResult{
			Identity:           ident,
			RateLimitLimit:     limitPerMin,
			RateLimitRemaining: 0,
			RateLimitResetSec:  retryAfter,
			QuotaLimit:         quotaLimit,
			QuotaRemaining:     quotaRemaining,
			QuotaResetSec:      secondsUntilMidnight,
			RetryAfterSec:      retryAfter,
		}
		return res, fmt.Errorf("%w: limit %d req/min for tier %q (retry in %ds)",
			ErrRateLimitExceeded, limitPerMin, tier.Name, retryAfter)
	}

	// Consume 1 token and increment quota
	bucket.tokens -= 1.0
	qRecord.count++
	m.stats.AllowedRequests++

	res := &CheckResult{
		Identity:           ident,
		RateLimitLimit:     limitPerMin,
		RateLimitRemaining: int(bucket.tokens),
		RateLimitResetSec:  int(math.Ceil((capacity - bucket.tokens) / fillRate)),
		QuotaLimit:         quotaLimit,
		QuotaRemaining:     quotaLimit - qRecord.count,
		QuotaResetSec:      secondsUntilMidnight,
	}

	return res, nil
}

// Stats returns a snapshot of current metrics.
func (m *Manager) Stats() Stats {
	m.mu.Lock()
	defer m.mu.Unlock()

	s := m.stats
	s.ActiveClients = len(m.buckets)
	return s
}

// Reset clears all in-memory usage tracking.
func (m *Manager) Reset() {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.buckets = make(map[string]*tokenBucket)
	m.quotas = make(map[string]*quotaRecord)
	m.stats = Stats{}
}

// extractAPIKey pulls an API key from Authorization header or X-API-Key header.
func extractAPIKey(r *http.Request) string {
	auth := strings.TrimSpace(r.Header.Get("Authorization"))
	if strings.HasPrefix(strings.ToLower(auth), "bearer ") {
		token := strings.TrimSpace(auth[7:])
		if token != "" {
			return token
		}
	}

	apiKey := strings.TrimSpace(r.Header.Get("X-API-Key"))
	if apiKey != "" {
		return apiKey
	}

	return ""
}

// extractClientIP extracts client IP address from proxy headers or RemoteAddr.
func extractClientIP(r *http.Request) string {
	if xff := r.Header.Get("X-Forwarded-For"); xff != "" {
		parts := strings.Split(xff, ",")
		clientIP := strings.TrimSpace(parts[0])
		if clientIP != "" {
			return clientIP
		}
	}

	if xrip := strings.TrimSpace(r.Header.Get("X-Real-IP")); xrip != "" {
		return xrip
	}

	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err == nil && host != "" {
		return host
	}

	return strings.TrimSpace(r.RemoteAddr)
}
