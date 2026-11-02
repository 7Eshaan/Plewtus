package orchestrator

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"testing"
	"time"

	"plewtus/internal/model"
	"plewtus/internal/provider"
	"plewtus/internal/provider/all"
)

// mockProvider implements provider.Provider for testing.
type mockProvider struct {
	name     string
	delay    time.Duration
	results  []model.RawSearchResult
	err      error
	panics   bool
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

	return &model.ProviderResponse{
		Engine:  m.name,
		Results: m.results,
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

	resp, err := orch.Search(context.Background(), &model.SearchRequest{
		Query:     "all fail",
		Providers: []string{"eng1", "eng2"},
	})

	if err != nil {
		t.Fatalf("unexpected search error: %v", err)
	}

	if len(resp.Results) != 0 {
		t.Errorf("expected 0 results, got %d", len(resp.Results))
	}

	if resp.Metadata.TotalResults != 0 {
		t.Errorf("expected TotalResults 0, got %d", resp.Metadata.TotalResults)
	}

	if len(resp.Metadata.Providers.Failed) != 2 {
		t.Errorf("expected 2 failed providers, got %v", resp.Metadata.Providers.Failed)
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
	if resp.Metadata.Stats.DedupedCount != 3 {
		t.Errorf("expected DedupedCount 3, got %d", resp.Metadata.Stats.DedupedCount)
	}
}

func TestOrchestrator_LiveEngines(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping live multi-provider orchestration test in short mode")
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
