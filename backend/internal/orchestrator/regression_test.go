package orchestrator

import (
	"context"
	"fmt"
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
