package server

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
	"time"

	"plewtus/internal/cache"
	"plewtus/internal/config"
	"plewtus/internal/model"
)

// mockTrackingSearcher tracks searcher call count.
type mockTrackingSearcher struct {
	calls int64
}

func (m *mockTrackingSearcher) Search(ctx context.Context, req *model.SearchRequest) (*model.SearchResponse, error) {
	atomic.AddInt64(&m.calls, 1)
	resp := model.NewSearchResponse(req.Query)
	resp.Results = []model.SearchResult{
		{
			ID:    "res-1",
			Title: "Test Result",
			URL:   "https://example.com/test",
			Rank:  1,
		},
	}
	resp.Metadata.TotalResults = 1
	return resp, nil
}

func TestServer_RateLimitingHeadersAnd429(t *testing.T) {
	cfg := config.Load()
	cfg.FreeTierRateLimit = 2 // Small limit for testing
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	searcher := &mockTrackingSearcher{}

	srv := New(cfg, logger, searcher)

	// 1st request: allowed
	req1 := httptest.NewRequest(http.MethodPost, "/search", bytes.NewBufferString(`{"query":"test rate limit"}`))
	req1.Header.Set("Content-Type", "application/json")
	req1.RemoteAddr = "192.168.1.50:5000"
	rec1 := httptest.NewRecorder()
	srv.Handler().ServeHTTP(rec1, req1)

	if rec1.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rec1.Code, rec1.Body.String())
	}
	if rec1.Header().Get("X-RateLimit-Limit") != "2" {
		t.Fatalf("expected X-RateLimit-Limit 2, got %s", rec1.Header().Get("X-RateLimit-Limit"))
	}
	if rec1.Header().Get("X-RateLimit-Remaining") != "1" {
		t.Fatalf("expected X-RateLimit-Remaining 1, got %s", rec1.Header().Get("X-RateLimit-Remaining"))
	}

	// 2nd request: allowed (remaining reaches 0)
	req2 := httptest.NewRequest(http.MethodPost, "/search", bytes.NewBufferString(`{"query":"test rate limit"}`))
	req2.Header.Set("Content-Type", "application/json")
	req2.RemoteAddr = "192.168.1.50:5000"
	rec2 := httptest.NewRecorder()
	srv.Handler().ServeHTTP(rec2, req2)

	if rec2.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec2.Code)
	}
	if rec2.Header().Get("X-RateLimit-Remaining") != "0" {
		t.Fatalf("expected X-RateLimit-Remaining 0, got %s", rec2.Header().Get("X-RateLimit-Remaining"))
	}

	// 3rd request: rate limit exceeded -> 429
	req3 := httptest.NewRequest(http.MethodPost, "/search", bytes.NewBufferString(`{"query":"test rate limit"}`))
	req3.Header.Set("Content-Type", "application/json")
	req3.RemoteAddr = "192.168.1.50:5000"
	rec3 := httptest.NewRecorder()
	srv.Handler().ServeHTTP(rec3, req3)

	if rec3.Code != http.StatusTooManyRequests {
		t.Fatalf("expected 429 Too Many Requests, got %d", rec3.Code)
	}
	if rec3.Header().Get("Retry-After") == "" {
		t.Fatal("expected Retry-After header on 429 response")
	}

	var errResp model.ErrorResponse
	if err := json.NewDecoder(rec3.Body).Decode(&errResp); err != nil {
		t.Fatalf("failed to decode error JSON: %v", err)
	}
	if errResp.Error != "rate_limit_exceeded" {
		t.Fatalf("expected error 'rate_limit_exceeded', got %s", errResp.Error)
	}
}

func TestServer_DailyQuotaExceeded429(t *testing.T) {
	cfg := config.Load()
	cfg.FreeTierRateLimit = 100 // High rate limit so quota trips first
	cfg.FreeTierDailyQuota = 2
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	searcher := &mockTrackingSearcher{}

	srv := New(cfg, logger, searcher)

	// Consume 2 requests
	for i := 1; i <= 2; i++ {
		req := httptest.NewRequest(http.MethodPost, "/search", bytes.NewBufferString(`{"query":"quota test"}`))
		req.Header.Set("Content-Type", "application/json")
		req.RemoteAddr = "10.10.10.10:1234"
		rec := httptest.NewRecorder()
		srv.Handler().ServeHTTP(rec, req)
		if rec.Code != http.StatusOK {
			t.Fatalf("request %d failed with status %d", i, rec.Code)
		}
	}

	// 3rd request: quota exceeded -> 429
	req3 := httptest.NewRequest(http.MethodPost, "/search", bytes.NewBufferString(`{"query":"quota test"}`))
	req3.Header.Set("Content-Type", "application/json")
	req3.RemoteAddr = "10.10.10.10:1234"
	rec3 := httptest.NewRecorder()
	srv.Handler().ServeHTTP(rec3, req3)

	if rec3.Code != http.StatusTooManyRequests {
		t.Fatalf("expected 429, got %d", rec3.Code)
	}

	var errResp model.ErrorResponse
	if err := json.NewDecoder(rec3.Body).Decode(&errResp); err != nil {
		t.Fatalf("failed to decode error JSON: %v", err)
	}
	if errResp.Error != "quota_exceeded" {
		t.Fatalf("expected error 'quota_exceeded', got %s", errResp.Error)
	}
}

func TestServer_Authentication401(t *testing.T) {
	cfg := config.Load()
	cfg.AuthRequired = true
	cfg.APIKeys = map[string]string{
		"valid-pro-key": "pro",
	}
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	searcher := &mockTrackingSearcher{}

	srv := New(cfg, logger, searcher)

	// 1. Missing API Key -> 401
	req1 := httptest.NewRequest(http.MethodPost, "/search", bytes.NewBufferString(`{"query":"auth test"}`))
	rec1 := httptest.NewRecorder()
	srv.Handler().ServeHTTP(rec1, req1)

	if rec1.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401 Unauthorized, got %d", rec1.Code)
	}

	// 2. Invalid API Key -> 401
	req2 := httptest.NewRequest(http.MethodPost, "/search", bytes.NewBufferString(`{"query":"auth test"}`))
	req2.Header.Set("Authorization", "Bearer invalid-token")
	rec2 := httptest.NewRecorder()
	srv.Handler().ServeHTTP(rec2, req2)

	if rec2.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401 Unauthorized, got %d", rec2.Code)
	}

	// 3. Valid Bearer Token -> 200
	req3 := httptest.NewRequest(http.MethodPost, "/search", bytes.NewBufferString(`{"query":"auth test"}`))
	req3.Header.Set("Authorization", "Bearer valid-pro-key")
	rec3 := httptest.NewRecorder()
	srv.Handler().ServeHTTP(rec3, req3)

	if rec3.Code != http.StatusOK {
		t.Fatalf("expected 200 OK with valid bearer key, got %d: %s", rec3.Code, rec3.Body.String())
	}
	if rec3.Header().Get("X-RateLimit-Limit") != "60" {
		t.Fatalf("expected Pro tier rate limit 60, got %s", rec3.Header().Get("X-RateLimit-Limit"))
	}

	// 4. Valid X-API-Key -> 200
	req4 := httptest.NewRequest(http.MethodPost, "/search", bytes.NewBufferString(`{"query":"auth test"}`))
	req4.Header.Set("X-API-Key", "valid-pro-key")
	rec4 := httptest.NewRecorder()
	srv.Handler().ServeHTTP(rec4, req4)

	if rec4.Code != http.StatusOK {
		t.Fatalf("expected 200 OK with X-API-Key, got %d", rec4.Code)
	}
}

// cachedSearcher wraps a searcher with a cache.
type cachedSearcher struct {
	base  Searcher
	cache cache.SearchCache
}

func (cs *cachedSearcher) Search(ctx context.Context, req *model.SearchRequest) (*model.SearchResponse, error) {
	key := cs.cache.Key(req)
	if hit, ok := cs.cache.Get(key); ok {
		return hit, nil
	}
	resp, err := cs.base.Search(ctx, req)
	if err != nil {
		return nil, err
	}
	cs.cache.Set(key, resp)
	return resp, nil
}

func TestServer_CacheIntegration(t *testing.T) {
	cfg := config.Load()
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	baseSearcher := &mockTrackingSearcher{}
	searchCache := cache.New(cache.Options{TTL: 5 * time.Minute, MaxEntries: 100})
	cSearcher := &cachedSearcher{base: baseSearcher, cache: searchCache}

	srv := New(cfg, logger, cSearcher)

	payload := `{"query":"caching server test"}`

	// 1st request -> searcher executed
	req1 := httptest.NewRequest(http.MethodPost, "/search", bytes.NewBufferString(payload))
	req1.Header.Set("Content-Type", "application/json")
	rec1 := httptest.NewRecorder()
	srv.Handler().ServeHTTP(rec1, req1)

	if rec1.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec1.Code)
	}
	var resp1 model.SearchResponse
	_ = json.NewDecoder(rec1.Body).Decode(&resp1)
	if resp1.Metadata.Cached {
		t.Fatal("expected first response not cached")
	}
	if atomic.LoadInt64(&baseSearcher.calls) != 1 {
		t.Fatalf("expected 1 call, got %d", atomic.LoadInt64(&baseSearcher.calls))
	}

	// 2nd request -> served from cache
	req2 := httptest.NewRequest(http.MethodPost, "/search", bytes.NewBufferString(payload))
	req2.Header.Set("Content-Type", "application/json")
	rec2 := httptest.NewRecorder()
	srv.Handler().ServeHTTP(rec2, req2)

	if rec2.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec2.Code)
	}
	var resp2 model.SearchResponse
	_ = json.NewDecoder(rec2.Body).Decode(&resp2)
	if !resp2.Metadata.Cached {
		t.Fatal("expected second response to have Metadata.Cached == true")
	}
	if atomic.LoadInt64(&baseSearcher.calls) != 1 {
		t.Fatalf("expected call count to remain 1 after cache hit, got %d", atomic.LoadInt64(&baseSearcher.calls))
	}
}
