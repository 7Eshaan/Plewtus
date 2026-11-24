package orchestrator

import (
	"context"
	"fmt"
	"sync/atomic"
	"testing"
	"time"

	"plewtus/internal/cache"
	"plewtus/internal/model"
	"plewtus/internal/provider"
)

// depthProvider ranks differently depending on how deep it is asked to go:
// one page returns pages 1-10 in order; two pages put ten new results first
// and return the first ten reversed, as a deeper fetch can reorder things.
type depthProvider struct{ calls int64 }

func (p *depthProvider) Name() string { return "depth" }
func (p *depthProvider) Search(ctx context.Context, req model.ProviderRequest) (*model.ProviderResponse, error) {
	atomic.AddInt64(&p.calls, 1)
	page := func(n int) model.RawSearchResult {
		return model.RawSearchResult{
			Title:   fmt.Sprintf("Golang concurrency patterns part %d", n),
			URL:     fmt.Sprintf("https://example.com/go-concurrency-%d", n),
			Snippet: "Golang concurrency patterns with goroutines and channels",
		}
	}
	var results []model.RawSearchResult
	if req.MaxPages <= 1 {
		for n := 1; n <= 10; n++ {
			results = append(results, page(n))
		}
	} else {
		for n := 11; n <= 20; n++ {
			results = append(results, page(n))
		}
		for n := 10; n >= 1; n-- {
			results = append(results, page(n))
		}
	}
	for i := range results {
		results[i].Position = i + 1
	}
	return &model.ProviderResponse{Engine: p.Name(), Results: results}, nil
}

func pageURLs(resp *model.SearchResponse) []string {
	urls := make([]string, len(resp.Results))
	for i, r := range resp.Results {
		urls[i] = r.URL
	}
	return urls
}

// Pages of one query never overlap or skip results, even when the next page
// needs a deeper fetch that ranks the first results differently.
func TestPagination_DeeperPageKeepsEarlierOrder(t *testing.T) {
	reg := provider.NewRegistry()
	p := &depthProvider{}
	reg.Register(p)
	o := New(reg, time.Second, "ua", testLogger()).WithCache(cache.New(cache.Options{TTL: time.Minute}))

	search := func(offset int) *model.SearchResponse {
		t.Helper()
		resp, err := o.Search(context.Background(), &model.SearchRequest{
			Query: "golang concurrency patterns", Limit: 10, Offset: offset, Providers: []string{"depth"},
		})
		if err != nil {
			t.Fatal(err)
		}
		return resp
	}

	first, second := search(0), search(10)
	shown := make(map[string]bool)
	for _, u := range pageURLs(first) {
		shown[u] = true
	}
	if len(first.Results) != 10 || len(second.Results) != 10 {
		t.Fatalf("expected two full pages, got %d and %d", len(first.Results), len(second.Results))
	}
	for i, r := range second.Results {
		if shown[r.URL] {
			t.Fatalf("page 2 repeats %s from page 1", r.URL)
		}
		if want := fmt.Sprintf("https://example.com/go-concurrency-%d", i+11); r.URL != want {
			t.Fatalf("page 2 result %d = %s, want %s (deeper results appended in their order)", i, r.URL, want)
		}
		if r.Rank != 11+i || r.ID != fmt.Sprintf("res_%d", 11+i) {
			t.Fatalf("page 2 result %d has rank %d / id %s, want %d / res_%d", i, r.Rank, r.ID, 11+i, 11+i)
		}
	}
	if second.Metadata.Stats.DedupedCount != 20 {
		t.Fatalf("expected 20 results available after the deeper fetch, got %d", second.Metadata.Stats.DedupedCount)
	}

	// Page 1 again (a new caller, same query) still comes back in the order
	// first shown, now served from the deeper list.
	again, err := o.Search(context.Background(), &model.SearchRequest{
		Query: "golang concurrency patterns", Limit: 5, Providers: []string{"depth"},
	})
	if err != nil {
		t.Fatal(err)
	}
	for i, r := range again.Results {
		if r.URL != first.Results[i].URL {
			t.Fatalf("page 1 changed order after the deeper fetch: %v vs %v", pageURLs(again), pageURLs(first))
		}
	}
	if got := atomic.LoadInt64(&p.calls); got != 2 {
		t.Fatalf("expected one provider fetch per depth (2), got %d", got)
	}
}

// Windows of the same depth share one cached list: paging inside the first
// ten results costs no further provider requests.
func TestPagination_SameDepthReusesList(t *testing.T) {
	reg := provider.NewRegistry()
	p := &depthProvider{}
	reg.Register(p)
	o := New(reg, time.Second, "ua", testLogger()).WithCache(cache.New(cache.Options{TTL: time.Minute}))

	var urls []string
	for _, offset := range []int{0, 5} {
		resp, err := o.Search(context.Background(), &model.SearchRequest{
			Query: "golang concurrency patterns", Limit: 5, Offset: offset, Providers: []string{"depth"},
		})
		if err != nil {
			t.Fatal(err)
		}
		if offset == 5 && !resp.Metadata.Cached {
			t.Fatal("the second page should be served from the cached list")
		}
		urls = append(urls, pageURLs(resp)...)
	}
	if got := atomic.LoadInt64(&p.calls); got != 1 {
		t.Fatalf("expected a single provider fetch, got %d", got)
	}
	for i, u := range urls {
		if want := fmt.Sprintf("https://example.com/go-concurrency-%d", i+1); u != want {
			t.Fatalf("result %d = %s, want %s", i, u, want)
		}
	}
}
