package orchestrator

import (
	"context"
	"errors"
	"sync/atomic"
	"testing"
	"time"

	"plewtus/internal/cache"
	"plewtus/internal/model"
	"plewtus/internal/provider"
	"plewtus/internal/reliability"
)

// countProvider tracks execution count.
type countProvider struct {
	name  string
	calls int64
	err   error
}

func (c *countProvider) Name() string { return c.name }
func (c *countProvider) Search(ctx context.Context, req model.ProviderRequest) (*model.ProviderResponse, error) {
	atomic.AddInt64(&c.calls, 1)
	if c.err != nil {
		return nil, c.err
	}
	return &model.ProviderResponse{
		Engine: c.name,
		Results: []model.RawSearchResult{
			{
				Title:   "Go Docs",
				URL:     "https://go.dev/doc",
				Snippet: "Documentation for Go " + req.Query,
			},
		},
	}, nil
}

func TestOrchestrator_CacheIntegration(t *testing.T) {
	reg := provider.NewRegistry()
	p := &countProvider{name: "mock1"}
	reg.Register(p)

	c := cache.New(cache.Options{TTL: 5 * time.Minute, MaxEntries: 100})
	orch := New(reg, 1*time.Second, "test", testLogger()).WithCache(c)

	req := &model.SearchRequest{
		Query:     "go tutorials",
		Limit:     10,
		Providers: []string{"mock1"},
	}

	// 1st search: cache miss
	resp1, err := orch.Search(context.Background(), req)
	if err != nil {
		t.Fatalf("first search failed: %v", err)
	}
	if resp1.Metadata.Cached {
		t.Fatal("expected first response not to be cached")
	}
	if atomic.LoadInt64(&p.calls) != 1 {
		t.Fatalf("expected 1 call on cache miss, got %d", atomic.LoadInt64(&p.calls))
	}

	// 2nd search: cache hit
	resp2, err := orch.Search(context.Background(), req)
	if err != nil {
		t.Fatalf("second search failed: %v", err)
	}
	if !resp2.Metadata.Cached {
		t.Fatal("expected second response to have Metadata.Cached == true")
	}
	if atomic.LoadInt64(&p.calls) != 1 {
		t.Fatalf("expected call count to remain 1 on cache hit, got %d", atomic.LoadInt64(&p.calls))
	}
	if len(resp2.Results) != 1 || resp2.Results[0].URL != "https://go.dev/doc" {
		t.Fatalf("unexpected cached results: %+v", resp2.Results)
	}
}

func TestOrchestrator_CircuitBreakerFastFail(t *testing.T) {
	reg := provider.NewRegistry()
	failingP := &countProvider{
		name: "fail-engine",
		err:  errors.New("simulated network error"),
	}
	reg.Register(failingP)

	breakers := reliability.NewManager(reliability.BreakerOptions{
		FailureThreshold: 2,
		Cooldown:         500 * time.Millisecond,
	})
	orch := New(reg, 1*time.Second, "test", testLogger()).WithBreakers(breakers)

	req := &model.SearchRequest{
		Query:     "circuit test",
		Providers: []string{"fail-engine"},
	}

	// First 2 calls fail and increment failure count to threshold
	_, _ = orch.Search(context.Background(), req)
	_, _ = orch.Search(context.Background(), req)

	if atomic.LoadInt64(&failingP.calls) != 2 {
		t.Fatalf("expected 2 calls, got %d", atomic.LoadInt64(&failingP.calls))
	}

	// Breaker should now be Open
	if breakers.IsHealthy("fail-engine") {
		t.Fatal("expected breaker to be open (unhealthy)")
	}

	// 3rd call should fast-fail without calling failingP
	resp3, err := orch.Search(context.Background(), req)
	if err != nil {
		t.Fatalf("expected Search to return partial response, got err: %v", err)
	}

	if atomic.LoadInt64(&failingP.calls) != 2 {
		t.Fatalf("expected provider NOT to be called when circuit is open, call count: %d", atomic.LoadInt64(&failingP.calls))
	}

	if len(resp3.Metadata.Providers.Failed) != 1 {
		t.Fatalf("expected 1 failed provider recorded, got %d", len(resp3.Metadata.Providers.Failed))
	}
	if resp3.Metadata.Providers.Failed[0].Engine != "fail-engine" {
		t.Fatalf("expected fail-engine recorded in failed, got %s", resp3.Metadata.Providers.Failed[0].Engine)
	}
}

func TestOrchestrator_CircuitBreakerFallback(t *testing.T) {
	reg := provider.NewRegistry()
	brokenP := &countProvider{name: "broken", err: errors.New("broken")}
	backupP := &countProvider{name: "backup"}
	reg.Register(brokenP)
	reg.Register(backupP)

	breakers := reliability.NewManager(reliability.BreakerOptions{
		FailureThreshold: 1,
		Cooldown:         1 * time.Second,
	})
	orch := New(reg, 1*time.Second, "test", testLogger()).WithBreakers(breakers)

	// Trip 'broken' circuit
	breakers.Get("broken").ReportFailure(errors.New("pre-trip"))

	// Query requesting only 'broken'
	req := &model.SearchRequest{
		Query:     "fallback test",
		Providers: []string{"broken"},
	}

	resp, err := orch.Search(context.Background(), req)
	if err != nil {
		t.Fatalf("search failed: %v", err)
	}

	// Backup provider should have been queried via fallback
	if atomic.LoadInt64(&backupP.calls) != 1 {
		t.Fatalf("expected fallback provider 'backup' to be called once, got %d", atomic.LoadInt64(&backupP.calls))
	}

	if len(resp.Results) != 1 {
		t.Fatalf("expected 1 result from fallback provider, got %d", len(resp.Results))
	}
	if resp.Results[0].Engines[0].Name != "backup" {
		t.Fatalf("expected backup provider in result engines, got %s", resp.Results[0].Engines[0].Name)
	}
	if len(resp.Metadata.Providers.Queried) != 1 || resp.Metadata.Providers.Queried[0] != "backup" {
		t.Fatalf("queried metadata should name the dispatched fallback only, got %v", resp.Metadata.Providers.Queried)
	}
	if len(resp.Metadata.Providers.Failed) != 1 || resp.Metadata.Providers.Failed[0].Engine != "broken" {
		t.Fatalf("breaker-skipped requested provider should be recorded as failed, got %+v", resp.Metadata.Providers.Failed)
	}
}
