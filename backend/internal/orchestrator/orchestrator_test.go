package orchestrator

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"os"
	"strings"
	"testing"
	"time"

	"plewtus/internal/model"
	"plewtus/internal/provider"
	"plewtus/internal/provider/all"
)

// mockProvider implements provider.Provider for testing.
type mockProvider struct {
	name    string
	delay   time.Duration
	results []model.RawSearchResult
	err     error
	panics  bool
}

func (m *mockProvider) Name() string {
	return m.name
}

func (m *mockProvider) Search(ctx context.Context, req model.ProviderRequest) (*model.ProviderResponse, error) {
	if m.panics {
		panic("simulated provider panic")
	}

	if m.delay > 0 {
		select {
		case <-time.After(m.delay):
		case <-ctx.Done():
			return nil, ctx.Err()
		}
	}

	if m.err != nil {
		return nil, m.err
	}

	results := append([]model.RawSearchResult(nil), m.results...)
	for i := range results {
		results[i].Snippet += " " + req.Query
	}
	return &model.ProviderResponse{
		Engine:  m.name,
		Results: results,
		Took:    m.delay,
	}, nil
}

func testLogger() *slog.Logger {
	return slog.New(slog.NewTextHandler(io.Discard, nil))
}

func TestOrchestrator_ParallelExecution(t *testing.T) {
	reg := provider.NewRegistry()
	delay := 100 * time.Millisecond

	reg.Register(&mockProvider{
		name:  "eng_a",
		delay: delay,
		results: []model.RawSearchResult{
			{Title: "A1", URL: "https://a.com/1", Position: 1},
		},
	})
	reg.Register(&mockProvider{
		name:  "eng_b",
		delay: delay,
		results: []model.RawSearchResult{
			{Title: "B1", URL: "https://b.com/1", Position: 1},
		},
	})
	reg.Register(&mockProvider{
		name:  "eng_c",
		delay: delay,
		results: []model.RawSearchResult{
			{Title: "C1", URL: "https://c.com/1", Position: 1},
		},
	})

	orch := New(reg, 2*time.Second, "test-agent", testLogger())

	start := time.Now()
	resp, err := orch.Search(context.Background(), &model.SearchRequest{
		Query:     "test parallel",
		Providers: []string{"eng_a", "eng_b", "eng_c"},
	})
	elapsed := time.Since(start)

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// 3 engines each took 100ms; in parallel, total elapsed should be ~100-200ms, definitely < 280ms
	if elapsed > 280*time.Millisecond {
		t.Errorf("expected parallel execution (<280ms), but took %v", elapsed)
	}

	if len(resp.Results) != 3 {
		t.Fatalf("expected 3 results, got %d", len(resp.Results))
	}

	if len(resp.Metadata.Providers.Successful) != 3 {
		t.Errorf("expected 3 successful providers, got %v", resp.Metadata.Providers.Successful)
	}

	if len(resp.Metadata.Providers.Failed) != 0 {
		t.Errorf("expected 0 failed providers, got %v", resp.Metadata.Providers.Failed)
	}

	if resp.Metadata.Stats.RawCount != 3 {
		t.Errorf("expected raw count 3, got %d", resp.Metadata.Stats.RawCount)
	}
}

func TestOrchestrator_PerProviderTimeout(t *testing.T) {
	reg := provider.NewRegistry()

	reg.Register(&mockProvider{
		name:  "fast",
		delay: 10 * time.Millisecond,
		results: []model.RawSearchResult{
			{Title: "Fast 1", URL: "https://fast.com/1", Position: 1},
		},
	})
	reg.Register(&mockProvider{
		name:  "slow",
		delay: 500 * time.Millisecond,
		results: []model.RawSearchResult{
			{Title: "Slow 1", URL: "https://slow.com/1", Position: 1},
		},
	})

	// Per-provider timeout is 60ms
	orch := New(reg, 60*time.Millisecond, "test-agent", testLogger())

	start := time.Now()
	resp, err := orch.Search(context.Background(), &model.SearchRequest{
		Query:     "test timeout",
		Providers: []string{"fast", "slow"},
	})
	elapsed := time.Since(start)

	if err != nil {
		t.Fatalf("unexpected search error: %v", err)
	}

	// Total duration should be near 60-150ms, not 500ms
	if elapsed > 350*time.Millisecond {
		t.Errorf("expected timeout enforcement (<350ms), got %v", elapsed)
	}

	if len(resp.Results) != 1 {
		t.Fatalf("expected 1 result from fast provider, got %d", len(resp.Results))
	}

	if len(resp.Metadata.Providers.Successful) != 1 || resp.Metadata.Providers.Successful[0] != "fast" {
		t.Errorf("expected ['fast'] successful, got %v", resp.Metadata.Providers.Successful)
	}

	if len(resp.Metadata.Providers.Failed) != 1 || resp.Metadata.Providers.Failed[0].Engine != "slow" {
		t.Errorf("expected ['slow'] failed, got %v", resp.Metadata.Providers.Failed)
	}
}

func TestOrchestrator_PartialResponseResilience(t *testing.T) {
	reg := provider.NewRegistry()

	reg.Register(&mockProvider{
		name: "working",
		results: []model.RawSearchResult{
			{Title: "W1", URL: "https://w.com/1", Position: 1},
			{Title: "W2", URL: "https://w.com/2", Position: 2},
		},
	})
	reg.Register(&mockProvider{
		name: "rate_limited",
		err:  errors.New("HTTP 429 Too Many Requests"),
	})
	reg.Register(&mockProvider{
		name:   "panicker",
		panics: true,
	})

	orch := New(reg, 1*time.Second, "test-agent", testLogger())

	resp, err := orch.Search(context.Background(), &model.SearchRequest{
		Query:     "test partial",
		Providers: []string{"working", "rate_limited", "panicker"},
	})

	if err != nil {
		t.Fatalf("unexpected search error: %v", err)
	}

	if len(resp.Results) != 2 {
		t.Fatalf("expected 2 results from working provider, got %d", len(resp.Results))
	}

	if len(resp.Metadata.Providers.Successful) != 1 || resp.Metadata.Providers.Successful[0] != "working" {
		t.Errorf("expected ['working'] successful, got %v", resp.Metadata.Providers.Successful)
	}

	if len(resp.Metadata.Providers.Failed) != 2 {
		t.Fatalf("expected 2 failed providers, got %d", len(resp.Metadata.Providers.Failed))
	}

	// Verify both failed engines are documented
	failedMap := make(map[string]string)
	for _, f := range resp.Metadata.Providers.Failed {
		failedMap[f.Engine] = f.Error
	}

	if _, ok := failedMap["rate_limited"]; !ok {
		t.Errorf("expected 'rate_limited' in failed providers")
	}
	if _, ok := failedMap["panicker"]; !ok {
		t.Errorf("expected 'panicker' in failed providers")
	}
}

func TestOrchestrator_UnknownProvider(t *testing.T) {
	reg := provider.NewRegistry()
	reg.Register(&mockProvider{
		name: "known",
		results: []model.RawSearchResult{
			{Title: "K1", URL: "https://k.com/1", Position: 1},
		},
	})

	orch := New(reg, 1*time.Second, "test-agent", testLogger())

	resp, err := orch.Search(context.Background(), &model.SearchRequest{
		Query:     "test unknown",
		Providers: []string{"known", "ghost_engine"},
	})

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(resp.Results) != 1 {
		t.Errorf("expected 1 result from known provider, got %d", len(resp.Results))
	}

	if len(resp.Metadata.Providers.Failed) != 1 || resp.Metadata.Providers.Failed[0].Engine != "ghost_engine" {
		t.Errorf("expected ghost_engine to be marked as failed, got %v", resp.Metadata.Providers.Failed)
	}
}

func TestOrchestrator_AllProvidersFail(t *testing.T) {
	reg := provider.NewRegistry()
	reg.Register(&mockProvider{
		name: "eng1",
		err:  errors.New("connection reset"),
	})
	reg.Register(&mockProvider{
		name: "eng2",
		err:  errors.New("503 service unavailable"),
	})

	orch := New(reg, 1*time.Second, "test-agent", testLogger())

	_, err := orch.Search(context.Background(), &model.SearchRequest{
		Query:     "all fail",
		Providers: []string{"eng1", "eng2"},
	})
	if err == nil || !strings.Contains(err.Error(), "all search providers failed") {
		t.Fatalf("expected explicit all-providers-failed error, got %v", err)
	}
}

func TestOrchestrator_ContextCancelled(t *testing.T) {
	reg := provider.NewRegistry()
	reg.Register(&mockProvider{
		name:  "sleeper",
		delay: 1 * time.Second,
	})

	orch := New(reg, 2*time.Second, "test-agent", testLogger())

	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel context immediately

	_, err := orch.Search(ctx, &model.SearchRequest{
		Query:     "cancelled",
		Providers: []string{"sleeper"},
	})

	if !errors.Is(err, context.Canceled) {
		t.Errorf("expected context.Canceled error, got %v", err)
	}
}

func TestOrchestrator_InterleavingAndLimit(t *testing.T) {
	reg := provider.NewRegistry()

	reg.Register(&mockProvider{
		name: "eng_a",
		results: []model.RawSearchResult{
			{Title: "A1", URL: "https://a.com/1", Position: 1},
			{Title: "A2", URL: "https://a.com/2", Position: 2},
			{Title: "A3", URL: "https://a.com/3", Position: 3},
		},
	})
	reg.Register(&mockProvider{
		name: "eng_b",
		results: []model.RawSearchResult{
			{Title: "B1", URL: "https://b.com/1", Position: 1},
			{Title: "B2", URL: "https://b.com/2", Position: 2},
			{Title: "B3", URL: "https://b.com/3", Position: 3},
		},
	})

	orch := New(reg, 1*time.Second, "test-agent", testLogger())

	resp, err := orch.Search(context.Background(), &model.SearchRequest{
		Query:     "interleaving",
		Limit:     3,
		Providers: []string{"eng_a", "eng_b"},
	})

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(resp.Results) != 3 {
		t.Fatalf("expected 3 results due to limit, got %d", len(resp.Results))
	}

	// Because eng_a < eng_b alphabetically:
	// pos 0: eng_a (A1), eng_b (B1)
	// pos 1: eng_a (A2) -> reaches limit of 3
	if resp.Results[0].Title != "A1" {
		t.Errorf("expected result 0 to be A1, got %s", resp.Results[0].Title)
	}
	if resp.Results[1].Title != "B1" {
		t.Errorf("expected result 1 to be B1, got %s", resp.Results[1].Title)
	}
	if resp.Results[2].Title != "A2" {
		t.Errorf("expected result 2 to be A2, got %s", resp.Results[2].Title)
	}

	if resp.Results[0].Rank != 1 || resp.Results[1].Rank != 2 || resp.Results[2].Rank != 3 {
		t.Errorf("expected sequential ranks 1, 2, 3")
	}

	if resp.Metadata.Stats.RawCount != 6 {
		t.Errorf("expected RawCount 6, got %d", resp.Metadata.Stats.RawCount)
	}
	if resp.Metadata.Stats.DedupedCount != 6 {
		t.Errorf("expected DedupedCount 6, got %d", resp.Metadata.Stats.DedupedCount)
	}
	if resp.Metadata.TotalResults != 3 {
		t.Errorf("expected TotalResults 3, got %d", resp.Metadata.TotalResults)
	}
}

func TestOrchestrator_Deduplication(t *testing.T) {
	reg := provider.NewRegistry()

	reg.Register(&mockProvider{
		name: "ddg",
		results: []model.RawSearchResult{
			{Title: "Go Lang", URL: "https://golang.org", Position: 1},
			{Title: "DDG Only", URL: "https://duckduckgo.com", Position: 2},
		},
	})
	reg.Register(&mockProvider{
		name: "bing",
		results: []model.RawSearchResult{
			{Title: "The Go Programming Language", URL: "https://www.golang.org/?utm_source=bing#intro", Position: 1},
			{Title: "Bing Only", URL: "https://bing.com", Position: 2},
		},
	})

	orch := New(reg, 1*time.Second, "test-agent", testLogger())

	resp, err := orch.Search(context.Background(), &model.SearchRequest{
		Query:     "go",
		Limit:     10,
		Providers: []string{"ddg", "bing"},
	})

	if err != nil {
		t.Fatalf("unexpected search error: %v", err)
	}

	// 4 raw results, but golang.org is duplicated -> 3 unique results
	if resp.Metadata.Stats.RawCount != 4 {
		t.Errorf("expected RawCount 4, got %d", resp.Metadata.Stats.RawCount)
	}
	if resp.Metadata.Stats.DedupedCount != 3 {
		t.Errorf("expected DedupedCount 3, got %d", resp.Metadata.Stats.DedupedCount)
	}
	if resp.Metadata.TotalResults != 3 {
		t.Errorf("expected TotalResults 3, got %d", resp.Metadata.TotalResults)
	}

	// Top result should be the duplicate URL confirmed by 2 engines
	first := resp.Results[0]
	if first.URL != "https://golang.org/" && first.URL != "https://www.golang.org/" {
		t.Errorf("expected golang.org as first result, got %s", first.URL)
	}
	if len(first.Engines) != 2 {
		t.Fatalf("expected 2 discovering engines for first result, got %d", len(first.Engines))
	}
	if first.Title != "The Go Programming Language" {
		t.Errorf("expected richer title, got %q", first.Title)
	}
}

func TestOrchestrator_LiveEngines(t *testing.T) {
	if testing.Short() || os.Getenv("PLEWTUS_LIVE_TESTS") != "1" {
		t.Skip("set PLEWTUS_LIVE_TESTS=1 to run live multi-provider orchestration test")
	}

	reg := all.NewRegistry(5*time.Second, "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/128.0.0.0 Safari/537.36")
	orch := New(reg, 6*time.Second, "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/128.0.0.0 Safari/537.36", testLogger())

	start := time.Now()
	resp, err := orch.Search(context.Background(), &model.SearchRequest{
		Query:     "golang concurrency patterns",
		Limit:     6,
		Providers: []string{"duckduckgo", "bing", "yahoo"},
	})
	elapsed := time.Since(start)

	if err != nil {
		t.Fatalf("live orchestrator search failed: %v", err)
	}

	t.Logf("Live parallel search completed in %v. Results: %d. Successful: %v. Failed: %v",
		elapsed, len(resp.Results), resp.Metadata.Providers.Successful, resp.Metadata.Providers.Failed)

	if len(resp.Metadata.Providers.Successful) == 0 {
		t.Fatalf("expected at least one live provider to succeed, failed: %v", resp.Metadata.Providers.Failed)
	}

	if len(resp.Results) == 0 {
		t.Fatalf("expected results from live engines, got 0")
	}

	for i, r := range resp.Results {
		if r.Title == "" || r.URL == "" {
			t.Errorf("result %d has empty title or url: %+v", i, r)
		}
		if r.Rank != i+1 {
			t.Errorf("result %d has rank %d, expected %d", i, r.Rank, i+1)
		}
	}
}

type mockExtractor struct {
	calledWithPages int
}

func (m *mockExtractor) Extract(ctx context.Context, targetURL string, opts *model.ExtractOptions) (*model.ExtractedContent, error) {
	return &model.ExtractedContent{Text: "mock text"}, nil
}

func (m *mockExtractor) ExtractBatch(ctx context.Context, results []model.SearchResult, opts *model.ExtractOptions) []model.SearchResult {
	limit := opts.MaxPages
	if limit > len(results) {
		limit = len(results)
	}
	m.calledWithPages = limit
	for i := 0; i < limit; i++ {
		results[i].Content = &model.ExtractedContent{
			Text:     "extracted: " + results[i].Title,
			Markdown: "# " + results[i].Title,
		}
	}
	return results
}

func TestOrchestrator_ContentExtractionIntegration(t *testing.T) {
	reg := provider.NewRegistry()
	reg.Register(&mockProvider{
		name: "test_engine",
		results: []model.RawSearchResult{
			{Title: "Page 1", URL: "https://example.com/1", Position: 1},
			{Title: "Page 2", URL: "https://example.com/2", Position: 2},
			{Title: "Page 3", URL: "https://example.com/3", Position: 3},
		},
	})

	mockExt := &mockExtractor{}
	orch := New(reg, 1*time.Second, "test-agent", testLogger()).WithExtractor(mockExt)

	// 1. Extraction enabled with MaxPages = 2
	resp, err := orch.Search(context.Background(), &model.SearchRequest{
		Query:     "test",
		Providers: []string{"test_engine"},
		Extract: &model.ExtractOptions{
			Enabled:  true,
			Mode:     "markdown",
			MaxPages: 2,
		},
	})

	if err != nil {
		t.Fatalf("unexpected search error: %v", err)
	}

	if mockExt.calledWithPages != 2 {
		t.Errorf("expected extractor to be called with 2 pages, got %d", mockExt.calledWithPages)
	}

	if resp.Results[0].Content == nil || resp.Results[0].Content.Markdown != "# Page 1" {
		t.Errorf("expected extracted markdown on result 0")
	}
	if resp.Results[1].Content == nil || resp.Results[1].Content.Markdown != "# Page 2" {
		t.Errorf("expected extracted markdown on result 1")
	}
	if resp.Results[2].Content != nil {
		t.Errorf("expected nil content on result 2 (exceeds MaxPages)")
	}
}

func TestOrchestrator_FiltersInMetadata(t *testing.T) {
	reg := provider.NewRegistry()
	reg.Register(&mockProvider{
		name: "test_engine",
		results: []model.RawSearchResult{
			{Title: "Page 1", URL: "https://go.dev/1", Position: 1},
			{Title: "Page 2", URL: "https://other.com/2", Position: 2},
		},
	})

	orch := New(reg, 1*time.Second, "test-agent", testLogger())

	resp, err := orch.Search(context.Background(), &model.SearchRequest{
		Query:          "test query",
		Freshness:      "month",
		Country:        "us",
		Language:       "en",
		IncludeDomains: []string{"go.dev"},
		Offset:         0,
		Limit:          5,
	})

	if err != nil {
		t.Fatalf("unexpected search error: %v", err)
	}

	if resp.Metadata.Filters == nil {
		t.Fatalf("expected metadata.filters to be populated, got nil")
	}

	f := resp.Metadata.Filters
	if f.Freshness != "month" {
		t.Errorf("expected freshness 'month', got %q", f.Freshness)
	}
	if f.Country != "us" {
		t.Errorf("expected country 'us', got %q", f.Country)
	}
	if f.Language != "en" {
		t.Errorf("expected language 'en', got %q", f.Language)
	}
	if len(f.IncludeDomains) != 1 || f.IncludeDomains[0] != "go.dev" {
		t.Errorf("expected include_domains ['go.dev'], got %v", f.IncludeDomains)
	}

	// Result from other.com must be filtered out
	if len(resp.Results) != 1 {
		t.Fatalf("expected 1 result from go.dev, got %d", len(resp.Results))
	}
	if resp.Results[0].URL != "https://go.dev/1" {
		t.Errorf("expected go.dev URL, got %s", resp.Results[0].URL)
	}
}

func TestOrchestrator_OffsetPagination(t *testing.T) {
	reg := provider.NewRegistry()
	reg.Register(&mockProvider{
		name: "test_engine",
		results: []model.RawSearchResult{
			{Title: "Item 1", URL: "https://example.com/1", Position: 1},
			{Title: "Item 2", URL: "https://example.com/2", Position: 2},
			{Title: "Item 3", URL: "https://example.com/3", Position: 3},
			{Title: "Item 4", URL: "https://example.com/4", Position: 4},
		},
	})

	orch := New(reg, 1*time.Second, "test-agent", testLogger())

	// Page 1: Offset 0, Limit 2
	respPage1, err := orch.Search(context.Background(), &model.SearchRequest{
		Query:  "test",
		Offset: 0,
		Limit:  2,
	})
	if err != nil {
		t.Fatalf("page 1 failed: %v", err)
	}
	if len(respPage1.Results) != 2 || respPage1.Results[0].Title != "Item 1" || respPage1.Results[1].Title != "Item 2" {
		t.Errorf("unexpected page 1 results: %+v", respPage1.Results)
	}

	// Page 2: Offset 2, Limit 2
	respPage2, err := orch.Search(context.Background(), &model.SearchRequest{
		Query:  "test",
		Offset: 2,
		Limit:  2,
	})
	if err != nil {
		t.Fatalf("page 2 failed: %v", err)
	}
	if len(respPage2.Results) != 2 || respPage2.Results[0].Title != "Item 3" || respPage2.Results[1].Title != "Item 4" {
		t.Errorf("unexpected page 2 results: %+v", respPage2.Results)
	}
}
