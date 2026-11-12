package orchestrator

import (
	"context"
	"fmt"
	"strings"
	"testing"
	"time"

	"plewtus/internal/cache"
	"plewtus/internal/model"
	"plewtus/internal/provider"
	"plewtus/internal/reliability"
)

// budgetExtractor records the deadline budget the extraction stage receives.
type budgetExtractor struct{ remaining chan time.Duration }

func (b *budgetExtractor) Extract(ctx context.Context, u string, o *model.ExtractOptions) (*model.ExtractedContent, error) {
	return nil, nil
}

func (b *budgetExtractor) ExtractBatch(ctx context.Context, rs []model.SearchResult, o *model.ExtractOptions) []model.SearchResult {
	if dl, ok := ctx.Deadline(); ok {
		b.remaining <- time.Until(dl)
	} else {
		b.remaining <- -1
	}
	return rs
}

// fixedProvider returns its results verbatim (mockProvider appends the query
// to every snippet, which would make every result look relevant).
type fixedProvider struct {
	name    string
	results []model.RawSearchResult
}

func (p *fixedProvider) Name() string { return p.name }
func (p *fixedProvider) Search(ctx context.Context, req model.ProviderRequest) (*model.ProviderResponse, error) {
	return &model.ProviderResponse{Engine: p.name, Results: append([]model.RawSearchResult(nil), p.results...)}, nil
}

func relevantResults() []model.RawSearchResult {
	return []model.RawSearchResult{{Title: "golang concurrency", URL: "https://go.dev/a", Snippet: "golang concurrency", Position: 1}}
}

// Regression: extraction must get its own time budget instead of whatever the
// provider stage left over, with and without the cache.
func TestRegression_ExtractionHasOwnBudget(t *testing.T) {
	for _, withCache := range []bool{false, true} {
		reg := provider.NewRegistry()
		reg.Register(&mockProvider{name: "slow", delay: 700 * time.Millisecond, results: relevantResults()})
		o := New(reg, time.Second, "ua", testLogger()).WithExtractTimeout(2 * time.Second)
		be := &budgetExtractor{remaining: make(chan time.Duration, 1)}
		o.WithExtractor(be)
		if withCache {
			o.WithCache(cache.New(cache.Options{TTL: time.Minute}))
		}
		_, err := o.Search(context.Background(), &model.SearchRequest{
			Query: "golang concurrency", Limit: 5, Providers: []string{"slow"},
			Extract: &model.ExtractOptions{Enabled: true, Mode: "markdown", MaxPages: 1},
		})
		if err != nil {
			t.Fatalf("cache=%v: %v", withCache, err)
		}
		if rem := <-be.remaining; rem < 1500*time.Millisecond {
			t.Fatalf("cache=%v: extraction budget %v, want close to 2s", withCache, rem)
		}
	}
}

// Regression: a caller cancelling must not count as a provider failure.
func TestRegression_ClientCancelDoesNotTripBreaker(t *testing.T) {
	reg := provider.NewRegistry()
	reg.Register(&mockProvider{name: "bing", delay: 300 * time.Millisecond, results: relevantResults()})
	o := New(reg, 5*time.Second, "ua", testLogger())
	mgr := reliability.NewManager(reliability.BreakerOptions{FailureThreshold: 2, Cooldown: time.Minute})
	o.WithBreakers(mgr)
	for i := 0; i < 3; i++ {
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Millisecond)
		_, _ = o.Search(ctx, &model.SearchRequest{Query: "golang concurrency", Limit: 5, Providers: []string{"bing"}})
		cancel()
	}
	time.Sleep(400 * time.Millisecond)
	if state := mgr.Get("bing").State(); state != reliability.StateClosed {
		t.Fatalf("breaker should stay closed after client cancellations, got %s", state)
	}
	if _, err := o.Search(context.Background(), &model.SearchRequest{Query: "golang concurrency", Limit: 5, Providers: []string{"bing"}}); err != nil {
		t.Fatalf("subsequent search failed: %v", err)
	}
}

// Regression: an empty result page is a valid answer, not a failure.
func TestRegression_ZeroResultsAreNotFailures(t *testing.T) {
	reg := provider.NewRegistry()
	reg.Register(&mockProvider{name: "bing", results: []model.RawSearchResult{}})
	o := New(reg, time.Second, "ua", testLogger())
	mgr := reliability.NewManager(reliability.BreakerOptions{FailureThreshold: 1, Cooldown: time.Minute})
	o.WithBreakers(mgr)
	resp, err := o.Search(context.Background(), &model.SearchRequest{Query: "site-restricted", Limit: 5, Providers: []string{"bing"}})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(resp.Metadata.Providers.Successful) != 1 || mgr.Get("bing").State() != reliability.StateClosed {
		t.Fatalf("empty response should be a success: %+v", resp.Metadata.Providers)
	}
}

// Regression: an off-topic (poisoned) SERP is reported as a failed provider.
func TestRegression_PoisonedProviderReportedAsFailed(t *testing.T) {
	var junk []model.RawSearchResult
	for i := 1; i <= 5; i++ {
		junk = append(junk, model.RawSearchResult{Title: fmt.Sprintf("PayPoint page %d", i), URL: fmt.Sprintf("https://junk.example/%d", i), Snippet: "login", Position: i})
	}
	reg := provider.NewRegistry()
	reg.Register(&fixedProvider{name: "bing", results: junk})
	reg.Register(&fixedProvider{name: "duckduckgo", results: relevantResults()})
	o := New(reg, time.Second, "ua", testLogger())
	resp, err := o.Search(context.Background(), &model.SearchRequest{Query: "golang concurrency", Limit: 10, Providers: []string{"bing", "duckduckgo"}})
	if err != nil {
		t.Fatal(err)
	}
	if len(resp.Results) != 1 || len(resp.Metadata.Providers.Failed) != 1 || resp.Metadata.Providers.Failed[0].Engine != "bing" {
		t.Fatalf("expected bing discarded and reported failed, got results=%d meta=%+v", len(resp.Results), resp.Metadata.Providers)
	}
	for _, s := range resp.Metadata.Providers.Successful {
		if s == "bing" {
			t.Fatal("poisoned engine must not be listed as successful")
		}
	}
}

func junkResults(n int) []model.RawSearchResult {
	var junk []model.RawSearchResult
	for i := 1; i <= n; i++ {
		junk = append(junk, model.RawSearchResult{Title: fmt.Sprintf("PayPoint page %d", i), URL: fmt.Sprintf("https://junk.example/%d", i), Snippet: "login", Position: i})
	}
	return junk
}

// Regression: a lone provider serving an unrelated page for a specific query
// yields no results (not the junk), and repeated junk opens its breaker.
func TestRegression_LonePoisonedProviderReturnsNothingAndTripsBreaker(t *testing.T) {
	reg := provider.NewRegistry()
	reg.Register(&fixedProvider{name: "bing", results: junkResults(6)})
	breakers := reliability.NewManager(reliability.BreakerOptions{FailureThreshold: 2, Cooldown: time.Minute})
	o := New(reg, time.Second, "ua", testLogger()).WithBreakers(breakers)
	req := func() *model.SearchRequest {
		return &model.SearchRequest{Query: "rust ownership borrow checker", Limit: 10, Providers: []string{"bing"}}
	}
	for i := 0; i < 2; i++ {
		resp, err := o.Search(context.Background(), req())
		if err != nil {
			t.Fatal(err)
		}
		if len(resp.Results) != 0 || len(resp.Metadata.Providers.Failed) != 1 || resp.Metadata.Stats.ProviderDiscardedCount != 6 {
			t.Fatalf("expected junk discarded, got results=%d meta=%+v", len(resp.Results), resp.Metadata)
		}
	}
	if breakers.Get("bing").State() != reliability.StateOpen {
		t.Fatalf("expected breaker open after repeated off-topic responses, got %s", breakers.Get("bing").State())
	}
}

// Regression: a healthy response closes the breaker (verdicts are reported
// after the quality gate, so this path must still report success).
func TestRegression_HealthyResponseReportsSuccess(t *testing.T) {
	reg := provider.NewRegistry()
	reg.Register(&fixedProvider{name: "duckduckgo", results: relevantResults()})
	breakers := reliability.NewManager(reliability.BreakerOptions{FailureThreshold: 2, Cooldown: time.Minute})
	breakers.Get("duckduckgo").ReportFailure(fmt.Errorf("earlier failure"))
	o := New(reg, time.Second, "ua", testLogger()).WithBreakers(breakers)
	if _, err := o.Search(context.Background(), &model.SearchRequest{Query: "golang concurrency", Limit: 10, Providers: []string{"duckduckgo"}}); err != nil {
		t.Fatal(err)
	}
	// One more failure must not trip the breaker: the success reset the count.
	breakers.Get("duckduckgo").ReportFailure(fmt.Errorf("later failure"))
	if breakers.Get("duckduckgo").State() != reliability.StateClosed {
		t.Fatal("expected success to reset the failure count")
	}
}

// Regression: a response missing a provider is cached only briefly.
func TestRegression_DegradedResponsesAreCachedBriefly(t *testing.T) {
	reg := provider.NewRegistry()
	reg.Register(&fixedProvider{name: "duckduckgo", results: relevantResults()})
	reg.Register(&mockProvider{name: "yahoo", err: fmt.Errorf("upstream down")})
	c := &ttlRecordingCache{MemoryCache: cache.New(cache.Options{TTL: 10 * time.Minute})}
	o := New(reg, time.Second, "ua", testLogger()).WithCache(c)
	if _, err := o.Search(context.Background(), &model.SearchRequest{Query: "golang concurrency", Limit: 10, Providers: []string{"duckduckgo", "yahoo"}}); err != nil {
		t.Fatal(err)
	}
	if c.lastTTL != degradedCacheTTL {
		t.Fatalf("expected degraded response cached for %s, got %s", degradedCacheTTL, c.lastTTL)
	}
}

type ttlRecordingCache struct {
	*cache.MemoryCache
	lastTTL time.Duration
}

func (c *ttlRecordingCache) Set(key string, resp *model.SearchResponse) {
	c.lastTTL = -1
	c.MemoryCache.Set(key, resp)
}

func (c *ttlRecordingCache) SetWithTTL(key string, resp *model.SearchResponse, ttl time.Duration) {
	c.lastTTL = ttl
	c.MemoryCache.SetWithTTL(key, resp, ttl)
}

// Regression: once the other engines have answered, a hung engine is cut off
// after the grace period instead of holding the response for its full timeout.
func TestRegression_StragglerIsCutOffAfterGrace(t *testing.T) {
	reg := provider.NewRegistry()
	reg.Register(&fixedProvider{name: "bing", results: relevantResults()})
	reg.Register(&fixedProvider{name: "duckduckgo", results: relevantResults()})
	reg.Register(&mockProvider{name: "yahoo", delay: 3 * time.Second, results: relevantResults()})
	breakers := reliability.NewManager(reliability.BreakerOptions{FailureThreshold: 1, Cooldown: time.Minute})
	o := New(reg, 5*time.Second, "ua", testLogger()).WithBreakers(breakers).WithStragglerGrace(50 * time.Millisecond)

	started := time.Now()
	resp, err := o.Search(context.Background(), &model.SearchRequest{Query: "golang concurrency", Limit: 10, Providers: []string{"bing", "duckduckgo", "yahoo"}})
	if err != nil {
		t.Fatal(err)
	}
	if elapsed := time.Since(started); elapsed > time.Second {
		t.Fatalf("expected the straggler to be cut off, search took %s", elapsed)
	}
	if len(resp.Metadata.Providers.Successful) != 2 || len(resp.Metadata.Providers.Failed) != 1 || resp.Metadata.Providers.Failed[0].Engine != "yahoo" {
		t.Fatalf("unexpected provider metadata: %+v", resp.Metadata.Providers)
	}
	time.Sleep(50 * time.Millisecond) // let the cancelled provider report
	if breakers.Get("yahoo").State() != reliability.StateClosed {
		t.Fatal("a cut-off straggler must not count as a provider failure")
	}
}

// Regression: providers are asked for further pages only for deep windows.
func TestRegression_DeepWindowsRequestMorePages(t *testing.T) {
	seen := make(chan model.ProviderRequest, 2)
	reg := provider.NewRegistry()
	reg.Register(&recordingProvider{seen: seen})
	o := New(reg, time.Second, "ua", testLogger())
	for _, req := range []*model.SearchRequest{
		{Query: "golang concurrency", Limit: 10, Providers: []string{"rec"}},
		{Query: "golang concurrency", Limit: 10, Offset: 20, Providers: []string{"rec"}},
	} {
		if _, err := o.Search(context.Background(), req); err != nil {
			t.Fatal(err)
		}
	}
	first, deep := <-seen, <-seen
	if first.MaxPages != 1 || first.Limit != 10 {
		t.Fatalf("default page should cost one request per engine, got %+v", first)
	}
	if deep.MaxPages != 3 || deep.Limit != 30 {
		t.Fatalf("offset 20 + limit 10 should request 30 results over 3 pages, got %+v", deep)
	}
}

type recordingProvider struct{ seen chan model.ProviderRequest }

func (p *recordingProvider) Name() string { return "rec" }
func (p *recordingProvider) Search(ctx context.Context, req model.ProviderRequest) (*model.ProviderResponse, error) {
	p.seen <- req
	return &model.ProviderResponse{Engine: "rec", Results: relevantResults()}, nil
}

// An engine that cannot express a filter must be reported as failed for that
// search without counting against its circuit breaker.
func TestUnsupportedFilterDoesNotTripBreaker(t *testing.T) {
	reg := provider.NewRegistry()
	reg.Register(&mockProvider{name: "picky", err: fmt.Errorf("%w: no weekly freshness", provider.ErrUnsupportedFilter)})
	reg.Register(&mockProvider{name: "ok", results: []model.RawSearchResult{{Title: "raft consensus", URL: "https://raft.github.io", Snippet: "raft", Position: 1}}})

	breakers := reliability.NewManager(reliability.BreakerOptions{FailureThreshold: 1, Cooldown: time.Minute})
	orch := New(reg, time.Second, "test", testLogger()).WithBreakers(breakers)

	for i := 0; i < 3; i++ {
		resp, err := orch.Search(context.Background(), &model.SearchRequest{Query: "raft", Limit: 5, Providers: []string{"picky", "ok"}})
		if err != nil {
			t.Fatalf("search %d failed: %v", i, err)
		}
		if len(resp.Metadata.Providers.Failed) != 1 || !strings.Contains(resp.Metadata.Providers.Failed[0].Error, "not supported") {
			t.Fatalf("search %d: expected picky reported as unsupported, got %+v", i, resp.Metadata.Providers.Failed)
		}
	}
	if state := breakers.Get("picky").State(); state != reliability.StateClosed {
		t.Fatalf("unsupported filters must not open the breaker, state=%s", state)
	}
}
