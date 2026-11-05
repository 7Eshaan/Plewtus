package usage

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
)

func TestUsage_FreeTierDefault(t *testing.T) {
	mgr := NewManager(Options{
		AuthRequired: false,
		Tiers: map[string]Tier{
			"free": {
				Name:            "free",
				RateLimitPerMin: 5,
				DailyQuota:      10,
			},
		},
	})

	req := httptest.NewRequest(http.MethodPost, "/search", nil)
	req.RemoteAddr = "192.0.2.1:12345"

	res, err := mgr.Check(req)
	if err != nil {
		t.Fatalf("expected anonymous free tier request allowed, got err: %v", err)
	}

	if res.Identity.Tier != "free" {
		t.Fatalf("expected tier 'free', got %s", res.Identity.Tier)
	}
	if res.Identity.Key != "ip:192.0.2.1" {
		t.Fatalf("expected key 'ip:192.0.2.1', got %s", res.Identity.Key)
	}
	if res.RateLimitLimit != 5 {
		t.Fatalf("expected rate limit 5, got %d", res.RateLimitLimit)
	}
	if res.RateLimitRemaining != 4 {
		t.Fatalf("expected rate limit remaining 4, got %d", res.RateLimitRemaining)
	}
	if res.QuotaRemaining != 9 {
		t.Fatalf("expected quota remaining 9, got %d", res.QuotaRemaining)
	}
}

func TestUsage_RateLimiting(t *testing.T) {
	mgr := NewManager(Options{
		AuthRequired: false,
		Tiers: map[string]Tier{
			"free": {
				Name:            "free",
				RateLimitPerMin: 3,
				DailyQuota:      100,
			},
		},
	})

	req := httptest.NewRequest(http.MethodPost, "/search", nil)
	req.RemoteAddr = "10.0.0.1:4321"

	// 3 requests allowed
	for i := 1; i <= 3; i++ {
		res, err := mgr.Check(req)
		if err != nil {
			t.Fatalf("request %d should have been allowed, got err: %v", i, err)
		}
		if res.RateLimitRemaining != 3-i {
			t.Fatalf("expected remaining %d, got %d", 3-i, res.RateLimitRemaining)
		}
	}

	// 4th request immediately after should be rate limited
	res, err := mgr.Check(req)
	if !errors.Is(err, ErrRateLimitExceeded) {
		t.Fatalf("expected ErrRateLimitExceeded, got: %v", err)
	}
	if res.RetryAfterSec <= 0 {
		t.Fatalf("expected positive RetryAfterSec, got %d", res.RetryAfterSec)
	}
	if res.RateLimitRemaining != 0 {
		t.Fatalf("expected remaining 0, got %d", res.RateLimitRemaining)
	}
}

func TestUsage_DailyQuota(t *testing.T) {
	mgr := NewManager(Options{
		AuthRequired: false,
		Tiers: map[string]Tier{
			"free": {
				Name:            "free",
				RateLimitPerMin: 100, // High rate limit to isolate quota test
				DailyQuota:      3,
			},
		},
	})

	req := httptest.NewRequest(http.MethodPost, "/search", nil)
	req.RemoteAddr = "172.16.0.5:8000"

	// Consume 3 quota units
	for i := 1; i <= 3; i++ {
		res, err := mgr.Check(req)
		if err != nil {
			t.Fatalf("request %d should succeed within quota, got: %v", i, err)
		}
		if res.QuotaRemaining != 3-i {
			t.Fatalf("expected quota remaining %d, got %d", 3-i, res.QuotaRemaining)
		}
	}

	// 4th request should exceed quota
	res, err := mgr.Check(req)
	if !errors.Is(err, ErrQuotaExceeded) {
		t.Fatalf("expected ErrQuotaExceeded, got: %v", err)
	}
	if res.QuotaRemaining != 0 {
		t.Fatalf("expected 0 quota remaining, got %d", res.QuotaRemaining)
	}
	if res.QuotaResetSec <= 0 {
		t.Fatalf("expected positive QuotaResetSec, got %d", res.QuotaResetSec)
	}
}

func TestUsage_APIKeyAuthentication(t *testing.T) {
	mgr := NewManager(Options{
		AuthRequired: true,
		Tiers: map[string]Tier{
			"free": {Name: "free", RateLimitPerMin: 10, DailyQuota: 100},
			"pro":  {Name: "pro", RateLimitPerMin: 60, DailyQuota: 2000},
		},
		APIKeys: map[string]string{
			"secret-pro-key": "pro",
		},
	})

	// 1. Missing key when auth required
	reqNoKey := httptest.NewRequest(http.MethodPost, "/search", nil)
	_, err := mgr.Check(reqNoKey)
	if !errors.Is(err, ErrAuthRequired) {
		t.Fatalf("expected ErrAuthRequired, got: %v", err)
	}

	// 2. Invalid key
	reqBadKey := httptest.NewRequest(http.MethodPost, "/search", nil)
	reqBadKey.Header.Set("Authorization", "Bearer invalid-token")
	_, err = mgr.Check(reqBadKey)
	if !errors.Is(err, ErrInvalidAPIKey) {
		t.Fatalf("expected ErrInvalidAPIKey, got: %v", err)
	}

	// 3. Valid key via Authorization: Bearer
	reqBearer := httptest.NewRequest(http.MethodPost, "/search", nil)
	reqBearer.Header.Set("Authorization", "Bearer secret-pro-key")
	resBearer, err := mgr.Check(reqBearer)
	if err != nil {
		t.Fatalf("expected bearer key allowed, got err: %v", err)
	}
	if resBearer.Identity.Tier != "pro" {
		t.Fatalf("expected tier 'pro', got %s", resBearer.Identity.Tier)
	}
	if resBearer.RateLimitLimit != 60 {
		t.Fatalf("expected pro limit 60, got %d", resBearer.RateLimitLimit)
	}

	// 4. Valid key via X-API-Key
	reqXKey := httptest.NewRequest(http.MethodPost, "/search", nil)
	reqXKey.Header.Set("X-API-Key", "secret-pro-key")
	resXKey, err := mgr.Check(reqXKey)
	if err != nil {
		t.Fatalf("expected X-API-Key allowed, got err: %v", err)
	}
	if resXKey.Identity.Tier != "pro" {
		t.Fatalf("expected tier 'pro', got %s", resXKey.Identity.Tier)
	}
}

func TestUsage_ClientIPExtraction(t *testing.T) {
	mgr := NewManager(Options{
		AuthRequired: false,
		Tiers: map[string]Tier{
			"free": {Name: "free", RateLimitPerMin: 10, DailyQuota: 100},
		},
	})

	// Forwarding headers from an untrusted socket peer must not change identity.
	req1 := httptest.NewRequest(http.MethodPost, "/search", nil)
	req1.RemoteAddr = "198.51.100.10:1234"
	req1.Header.Set("X-Forwarded-For", "203.0.113.195, 70.41.3.18, 150.172.238.178")
	req1.Header.Set("X-Real-IP", "203.0.113.196")
	res1, err := mgr.Check(req1)
	if err != nil || res1.Identity.Address != "198.51.100.10" {
		t.Fatalf("untrusted forwarding headers changed client identity: result=%+v err=%v", res1, err)
	}

	// A configured trusted proxy may provide an X-Forwarded-For chain.
	trustedMgr := NewManager(Options{
		Tiers:          map[string]Tier{"free": {Name: "free", RateLimitPerMin: 10, DailyQuota: 100}},
		TrustedProxies: []string{"10.0.0.0/8"},
	})
	req2 := httptest.NewRequest(http.MethodPost, "/search", nil)
	req2.RemoteAddr = "10.0.0.2:443"
	req2.Header.Set("X-Forwarded-For", "203.0.113.195, 10.0.0.1")
	res2, err := trustedMgr.Check(req2)
	if err != nil || res2.Identity.Address != "203.0.113.195" {
		t.Fatalf("expected trusted proxy client address 203.0.113.195, result=%+v err=%v", res2, err)
	}

	// RemoteAddr with IPv6 and port is normalized when no trusted proxy is configured.
	req3 := httptest.NewRequest(http.MethodPost, "/search", nil)
	req3.RemoteAddr = "[2001:db8::1]:54321"
	res3, _ := mgr.Check(req3)
	if res3.Identity.Address != "2001:db8::1" {
		t.Fatalf("expected IPv6 '2001:db8::1', got %s", res3.Identity.Address)
	}
}

func TestUsage_ConcurrencyAndStats(t *testing.T) {
	mgr := NewManager(Options{
		AuthRequired: false,
		Tiers: map[string]Tier{
			"free": {
				Name:            "free",
				RateLimitPerMin: 1000,
				DailyQuota:      10000,
			},
		},
	})

	var wg sync.WaitGroup
	for i := 0; i < 20; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			req := httptest.NewRequest(http.MethodPost, "/search", nil)
			req.RemoteAddr = "10.0.0.100:1234"
			for j := 0; j < 50; j++ {
				_, _ = mgr.Check(req)
			}
		}(i)
	}
	wg.Wait()

	stats := mgr.Stats()
	if stats.TotalRequests != 1000 {
		t.Fatalf("expected 1000 total requests, got %d", stats.TotalRequests)
	}
	if stats.AllowedRequests != 1000 {
		t.Fatalf("expected 1000 allowed requests, got %d", stats.AllowedRequests)
	}
}
