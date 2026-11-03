package test

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"plewtus/internal/cache"
	"plewtus/internal/config"
	"plewtus/internal/model"
	"plewtus/internal/orchestrator"
	"plewtus/internal/provider"
	"plewtus/internal/reliability"
	"plewtus/internal/server"
	"plewtus/internal/usage"
)

// e2eProvider simulates an external search engine returning raw search results with tracking parameters.
type e2eProvider struct {
	name    string
	results []model.RawSearchResult
	err     error
}

func (p *e2eProvider) Name() string { return p.name }
func (p *e2eProvider) Search(ctx context.Context, req model.ProviderRequest) (*model.ProviderResponse, error) {
	if p.err != nil {
		return nil, p.err
	}
	return &model.ProviderResponse{
		Engine:  p.name,
		Results: p.results,
		Took:    10 * time.Millisecond,
	}, nil
}

// e2eExtractor simulates webpage content extraction.
type e2eExtractor struct{}

func (e *e2eExtractor) Extract(ctx context.Context, targetURL string, opts *model.ExtractOptions) (*model.ExtractedContent, error) {
	return &model.ExtractedContent{
		Markdown:    fmt.Sprintf("# Extracted: %s\nClean article content.", targetURL),
		ExtractedAt: time.Now().UTC().Format(time.RFC3339),
	}, nil
}

func (e *e2eExtractor) ExtractBatch(ctx context.Context, results []model.SearchResult, opts *model.ExtractOptions) []model.SearchResult {
	for i := range results {
		content, _ := e.Extract(ctx, results[i].URL, opts)
		results[i].Content = content
	}
	return results
}

func TestE2E_FullPipelineLifecycle(t *testing.T) {
	// 1. Setup multi-engine mock providers with tracking parameters and duplicate URLs
	registry := provider.NewRegistry()

	// Provider 1: DuckDuckGo discovers A, B
	registry.Register(&e2eProvider{
		name: "duckduckgo",
		results: []model.RawSearchResult{
			{
				Title:   "Go Official Site",
				URL:     "https://go.dev/?utm_source=ddg&utm_medium=cpc",
				Snippet: "Build simple, secure, scalable systems with Go.",
			},
			{
				Title:   "Go Tour",
				URL:     "https://tour.golang.org/#page1",
				Snippet: "An interactive tour of Go.",
			},
		},
	})

	// Provider 2: Bing discovers A (duplicate canonical with www and tracking parameters), C
	registry.Register(&e2eProvider{
		name: "bing",
		results: []model.RawSearchResult{
			{
				Title:   "The Go Programming Language",
				URL:     "https://www.go.dev/?msclkid=123456",
				Snippet: "The official home of the Go programming language.",
			},
			{
				Title:   "Go by Example",
				URL:     "https://gobyexample.com/?ref=search",
				Snippet: "Hands-on Go examples.",
			},
		},
	})

	// Provider 3: Yahoo discovers A (duplicate canonical), D
	registry.Register(&e2eProvider{
		name: "yahoo",
		results: []model.RawSearchResult{
			{
				Title:   "Go Dev Home",
				URL:     "http://go.dev/index.html",
				Snippet: "Go open source programming language.",
			},
			{
				Title:   "Go Packages",
				URL:     "https://pkg.go.dev/",
				Snippet: "Find Go packages and documentation.",
			},
		},
	})

	// 2. Setup Orchestrator, Cache, Circuit Breakers, and Extractor
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	timeout := 2 * time.Second

	searchCache := cache.New(cache.Options{TTL: 5 * time.Minute, MaxEntries: 100})
	breakers := reliability.NewManager(reliability.BreakerOptions{
		FailureThreshold: 2,
		Cooldown:         1 * time.Second,
	})

	orch := orchestrator.New(registry, timeout, "plewtus-test", logger)
	orch.WithCache(searchCache)
	orch.WithBreakers(breakers)
	orch.WithExtractor(&e2eExtractor{})

	// 3. Setup Usage Management and Server
	cfg := config.Load()
	cfg.AuthRequired = true
	cfg.FreeTierRateLimit = 10
	cfg.FreeTierDailyQuota = 50
	cfg.ProTierRateLimit = 60
	cfg.ProTierDailyQuota = 2000

	usageMgr := usage.NewManager(usage.Options{
		AuthRequired: true,
		Tiers: map[string]usage.Tier{
			"pro": {Name: "pro", RateLimitPerMin: 60, DailyQuota: 2000},
		},
		APIKeys: map[string]string{
			"pro-api-key": "pro",
		},
	})

	srv := server.New(cfg, logger, orch).WithUsageManager(usageMgr)
	handler := srv.Handler()

	// ----------------------------------------------------
	// STEP A: Verify Authentication Guard (401 without key)
	// ----------------------------------------------------
	reqNoAuth := httptest.NewRequest(http.MethodPost, "/search", strings.NewReader(`{"query":"golang"}`))
	reqNoAuth.Header.Set("Content-Type", "application/json")
	recNoAuth := httptest.NewRecorder()
	handler.ServeHTTP(recNoAuth, reqNoAuth)

	if recNoAuth.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401 Unauthorized without API key, got %d", recNoAuth.Code)
	}

	// ----------------------------------------------------
	// STEP B: Successful Authenticated Search with Extraction & RRF
	// ----------------------------------------------------
	reqBody := `{
		"query": "golang",
		"limit": 5,
		"extract": {
			"enabled": true,
			"mode": "markdown"
		}
	}`
	reqAuth := httptest.NewRequest(http.MethodPost, "/search", strings.NewReader(reqBody))
	reqAuth.Header.Set("Content-Type", "application/json")
	reqAuth.Header.Set("Authorization", "Bearer pro-api-key")
	recAuth := httptest.NewRecorder()
	handler.ServeHTTP(recAuth, reqAuth)

	if recAuth.Code != http.StatusOK {
		t.Fatalf("expected 200 OK, got %d: %s", recAuth.Code, recAuth.Body.String())
	}

	// Verify Rate Limit & Quota Headers
	if recAuth.Header().Get("X-RateLimit-Limit") != "60" {
		t.Fatalf("expected X-RateLimit-Limit 60, got %s", recAuth.Header().Get("X-RateLimit-Limit"))
	}
	if recAuth.Header().Get("X-Quota-Limit") != "2000" {
		t.Fatalf("expected X-Quota-Limit 2000, got %s", recAuth.Header().Get("X-Quota-Limit"))
	}

	var searchResp model.SearchResponse
	if err := json.NewDecoder(recAuth.Body).Decode(&searchResp); err != nil {
		t.Fatalf("failed to decode SearchResponse: %v", err)
	}

	// Verify query echoed
	if searchResp.Query != "golang" {
		t.Fatalf("expected echoed query 'golang', got %s", searchResp.Query)
	}

	// Verify Deduplication & Multi-Engine Consensus:
	// https://go.dev/ was returned by all 3 engines with tracking parameters stripped.
	// It must be ranked #1 with 3 engine sources!
	if len(searchResp.Results) == 0 {
		t.Fatal("expected search results, got 0")
	}

	topResult := searchResp.Results[0]
	if !strings.Contains(topResult.URL, "go.dev") {
		t.Fatalf("expected top URL to contain 'go.dev', got %s", topResult.URL)
	}
	if len(topResult.Engines) != 3 {
		t.Fatalf("expected 3 discovering engines for top result, got %d", len(topResult.Engines))
	}

	// Formula check: RRF score = 1/(60+1) + 1/(60+1) + 1/(60+1) = 3/61 ≈ 0.049180
	if topResult.Score < 0.049 || topResult.Score > 0.050 {
		t.Fatalf("expected RRF score ≈ 0.049180, got %f", topResult.Score)
	}

	// Verify Content Extraction
	if topResult.Content == nil || !strings.Contains(topResult.Content.Markdown, "Extracted:") {
		t.Fatalf("expected extracted Markdown content, got: %+v", topResult.Content)
	}

	// Verify Metadata
	if searchResp.Metadata.Stats.RawCount != 6 {
		t.Fatalf("expected raw_count 6, got %d", searchResp.Metadata.Stats.RawCount)
	}
	if searchResp.Metadata.Stats.DedupedCount != 4 {
		t.Fatalf("expected deduped_count 4, got %d", searchResp.Metadata.Stats.DedupedCount)
	}
	if searchResp.Metadata.Cached {
		t.Fatal("expected first request not to be cached")
	}

	// ----------------------------------------------------
	// STEP C: In-Memory Cache Verification on Repeated Query
	// ----------------------------------------------------
	reqCache := httptest.NewRequest(http.MethodPost, "/search", strings.NewReader(reqBody))
	reqCache.Header.Set("Content-Type", "application/json")
	reqCache.Header.Set("Authorization", "Bearer pro-api-key")
	recCache := httptest.NewRecorder()
	handler.ServeHTTP(recCache, reqCache)

	if recCache.Code != http.StatusOK {
		t.Fatalf("expected 200 OK on cached query, got %d", recCache.Code)
	}

	var cachedResp model.SearchResponse
	if err := json.NewDecoder(recCache.Body).Decode(&cachedResp); err != nil {
		t.Fatalf("failed to decode cached SearchResponse: %v", err)
	}
	if !cachedResp.Metadata.Cached {
		t.Fatal("expected second identical request to return Metadata.Cached == true")
	}
	if len(cachedResp.Results) != len(searchResp.Results) {
		t.Fatalf("cached result count mismatch: %d vs %d", len(cachedResp.Results), len(searchResp.Results))
	}
}

func TestE2E_CircuitBreakerAndFallback(t *testing.T) {
	registry := provider.NewRegistry()
	registry.Register(&e2eProvider{
		name: "flaky-primary",
		err:  fmt.Errorf("upstream gateway timeout"),
	})
	registry.Register(&e2eProvider{
		name: "backup-engine",
		results: []model.RawSearchResult{
			{Title: "Backup Result", URL: "https://backup.example.com", Snippet: "From backup"},
		},
	})

	breakers := reliability.NewManager(reliability.BreakerOptions{
		FailureThreshold: 2,
		Cooldown:         500 * time.Millisecond,
	})

	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	orch := orchestrator.New(registry, 1*time.Second, "plewtus-test", logger)
	orch.WithBreakers(breakers)

	cfg := config.Load()
	cfg.AuthRequired = false // Allow free tier
	srv := server.New(cfg, logger, orch)

	// Trip flaky-primary twice
	for i := 0; i < 2; i++ {
		req := httptest.NewRequest(http.MethodPost, "/search", bytes.NewBufferString(`{"query":"test","providers":["flaky-primary"]}`))
		req.Header.Set("Content-Type", "application/json")
		rec := httptest.NewRecorder()
		srv.Handler().ServeHTTP(rec, req)
	}

	// 3rd request requesting flaky-primary: breaker is open, should fallback to backup-engine
	reqFallback := httptest.NewRequest(http.MethodPost, "/search", bytes.NewBufferString(`{"query":"test","providers":["flaky-primary"]}`))
	reqFallback.Header.Set("Content-Type", "application/json")
	recFallback := httptest.NewRecorder()
	srv.Handler().ServeHTTP(recFallback, reqFallback)

	if recFallback.Code != http.StatusOK {
		t.Fatalf("expected 200 OK via circuit breaker fallback, got %d", recFallback.Code)
	}

	var resp model.SearchResponse
	_ = json.NewDecoder(recFallback.Body).Decode(&resp)

	if len(resp.Results) != 1 || resp.Results[0].URL != "https://backup.example.com/" {
		t.Fatalf("expected fallback result from backup-engine, got: %+v", resp.Results)
	}
	if len(resp.Metadata.Providers.Failed) != 1 || !strings.Contains(resp.Metadata.Providers.Failed[0].Error, "circuit breaker") {
		t.Fatalf("expected flaky-primary recorded as failed with circuit breaker notice: %+v", resp.Metadata.Providers.Failed)
	}
}
