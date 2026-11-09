package usage

import (
	"errors"
	"fmt"
	"math"
	"net"
	"net/http"
	"net/netip"
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
	Identity           Identity
	RateLimitLimit     int
	RateLimitRemaining int
	RateLimitResetSec  int
	QuotaLimit         int
	QuotaRemaining     int
	QuotaResetSec      int
	RetryAfterSec      int
}

// Options configures the UsageManager.
type Options struct {
	AuthRequired   bool              // If true, requests without a valid API key are rejected (401)
	Tiers          map[string]Tier   // Limits per tier
	APIKeys        map[string]string // map[apiKey]tierName
	TrustedProxies []string          // CIDRs or IP addresses allowed to supply forwarding headers
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
	mu             sync.Mutex
	authRequired   bool
	tiers          map[string]Tier
	apiKeys        map[string]string // apiKey -> tierName
	trustedProxies []netip.Prefix
	buckets        map[string]*tokenBucket
	quotas         map[string]*quotaRecord
	stats          Stats
	lastCleanup    time.Time
}

const (
	maxTrackedClients = 50000
	usageRecordTTL    = 24 * time.Hour
)

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
	var trustedProxies []netip.Prefix
	for _, raw := range opts.TrustedProxies {
		raw = strings.TrimSpace(raw)
		if prefix, err := netip.ParsePrefix(raw); err == nil {
			trustedProxies = append(trustedProxies, prefix.Masked())
			continue
		}
		if addr, err := netip.ParseAddr(raw); err == nil {
			trustedProxies = append(trustedProxies, netip.PrefixFrom(addr, addr.BitLen()))
		}
	}

	return &Manager{
		authRequired:   opts.AuthRequired,
		tiers:          tiers,
		apiKeys:        apiKeys,
		trustedProxies: trustedProxies,
		buckets:        make(map[string]*tokenBucket),
		quotas:         make(map[string]*quotaRecord),
		lastCleanup:    time.Now(),
	}
}

// Authenticate verifies the caller's credentials without consuming rate-limit
// tokens or quota, so handlers can reject unauthenticated calls before parsing.
func (m *Manager) Authenticate(r *http.Request) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if rawKey := extractAPIKey(r); rawKey != "" {
		if _, ok := m.apiKeys[rawKey]; !ok {
			return ErrInvalidAPIKey
		}
		return nil
	}
	if m.authRequired {
		return ErrAuthRequired
	}
	return nil
}

// Check inspects the incoming request, authenticates, and validates rate limit and quota.
func (m *Manager) Check(r *http.Request) (*CheckResult, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	now := time.Now().UTC()
	m.cleanupStaleRecords(now)
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
			Address: m.extractClientIP(r),
		}
	} else {
		if m.authRequired {
			m.stats.UnauthorizedCount++
			return nil, ErrAuthRequired
		}
		ip := m.extractClientIP(r)
		ident = Identity{
			Key:     "ip:" + ip,
			Type:    "ip",
			Tier:    "free",
			Address: ip,
		}
	}

	// Bound memory use if a host serves more unique anonymous addresses than
	// this in-memory manager can track. Overflow clients share one free-tier bucket.
	if ident.Type == "ip" {
		if _, tracked := m.buckets[ident.Key]; !tracked && len(m.buckets) >= maxTrackedClients {
			ident.Key = "ip:overflow"
		}
	}

	tier, exists := m.tiers[ident.Tier]
	if !exists {
		// Fallback to free tier limits if tier configuration is missing
		tier = m.tiers["free"]
	}

	// 2. Daily Quota Check
	today := now.Format("2006-01-02")
	secondsUntilMidnight := int(now.Truncate(24 * time.Hour).Add(24 * time.Hour).Sub(now).Seconds())

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
			Identity:       ident,
			RateLimitLimit: tier.RateLimitPerMin,
			QuotaLimit:     quotaLimit,
			QuotaRemaining: 0,
			QuotaResetSec:  secondsUntilMidnight,
			RetryAfterSec:  secondsUntilMidnight,
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

func (m *Manager) cleanupStaleRecords(now time.Time) {
	if now.Sub(m.lastCleanup) < time.Minute {
		return
	}
	m.lastCleanup = now
	cutoff := now.Add(-usageRecordTTL)
	for key, bucket := range m.buckets {
		if bucket.lastRefill.Before(cutoff) {
			delete(m.buckets, key)
			delete(m.quotas, key)
		}
	}
	// Quota records can exist without a bucket (e.g. callers rejected for an
	// exhausted quota); drop any from previous UTC days.
	today := now.Format("2006-01-02")
	for key, record := range m.quotas {
		if record.day != today {
			if _, hasBucket := m.buckets[key]; !hasBucket {
				delete(m.quotas, key)
			}
		}
	}
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
func (m *Manager) extractClientIP(r *http.Request) string {
	remoteHost, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		remoteHost = strings.TrimSpace(r.RemoteAddr)
	}
	remoteAddr, err := netip.ParseAddr(remoteHost)
	if err != nil || !m.isTrustedProxy(remoteAddr) {
		return remoteHost
	}

	// Walk the forwarded chain from the trusted server-facing proxy toward the
	// client. Ignore trusted proxy hops and stop at the first untrusted address.
	parts := strings.Split(r.Header.Get("X-Forwarded-For"), ",")
	for i := len(parts) - 1; i >= 0; i-- {
		candidate, err := netip.ParseAddr(strings.TrimSpace(parts[i]))
		if err != nil {
			continue
		}
		if !m.isTrustedProxy(candidate) {
			return candidate.String()
		}
	}
	return remoteAddr.String()
}

func (m *Manager) isTrustedProxy(addr netip.Addr) bool {
	for _, prefix := range m.trustedProxies {
		if prefix.Contains(addr) {
			return true
		}
	}
	return false
}
