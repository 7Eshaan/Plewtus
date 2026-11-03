package cache

import (
	"fmt"
	"sync"
	"testing"
	"time"

	"plewtus/internal/model"
)

func TestCache_KeyDeterminism(t *testing.T) {
	c := New(Options{TTL: 5 * time.Minute, MaxEntries: 100})

	req1 := &model.SearchRequest{
		Query:          "golang metasearch",
		Limit:          10,
		Offset:         0,
		Providers:      []string{"duckduckgo", "bing", "yahoo"},
		IncludeDomains: []string{"go.dev", "golang.org"},
		ExcludeDomains: []string{"pinterest.com"},
		Freshness:      "week",
		Country:        "us",
		Language:       "en",
		Extract: &model.ExtractOptions{
			Enabled:         true,
			Mode:            "markdown",
			MaxPages:        3,
			MaxCharsPerPage: 4000,
		},
	}

	// req2 has permuted provider and domain orders, plus whitespace/casing variations
	req2 := &model.SearchRequest{
		Query:          "  golang metasearch  ",
		Limit:          10,
		Offset:         0,
		Providers:      []string{"yahoo", "duckduckgo", "bing"},
		IncludeDomains: []string{"golang.org", "go.dev"},
		ExcludeDomains: []string{"pinterest.com"},
		Freshness:      "WEEK",
		Country:        "US",
		Language:       "EN",
		Extract: &model.ExtractOptions{
			Enabled:         true,
			Mode:            "MARKDOWN",
			MaxPages:        3,
			MaxCharsPerPage: 4000,
		},
	}

	key1 := c.Key(req1)
	key2 := c.Key(req2)

	if key1 == "" {
		t.Fatal("expected non-empty cache key")
	}
	if key1 != key2 {
		t.Fatalf("expected identical keys for equivalent requests, got key1=%s, key2=%s", key1, key2)
	}

	// req3 has different query
	req3 := &model.SearchRequest{
		Query: "python metasearch",
		Limit: 10,
	}
	key3 := c.Key(req3)
	if key1 == key3 {
		t.Fatalf("expected different keys for different queries, got both=%s", key1)
	}
}

func TestCache_GetSetAndCloneIsolation(t *testing.T) {
	c := New(Options{TTL: 10 * time.Minute, MaxEntries: 10})

	key := "test-key-1"
	original := model.NewSearchResponse("test query")
	original.Results = []model.SearchResult{
		{
			ID:      "res-1",
			Title:   "Go Language",
			URL:     "https://go.dev",
			Snippet: "Build simple, secure, scalable systems.",
			Rank:    1,
			Score:   0.016393,
			Engines: []model.EngineSource{{Name: "duckduckgo", Position: 1}},
			Content: &model.ExtractedContent{
				Text:       "Go is an open source programming language.",
				Highlights: []string{"open source"},
			},
		},
	}
	original.Metadata.TotalResults = 1

	c.Set(key, original)
	if c.Len() != 1 {
		t.Fatalf("expected cache len=1, got %d", c.Len())
	}

	cached, found := c.Get(key)
	if !found {
		t.Fatal("expected item to be found in cache")
	}

	if !cached.Metadata.Cached {
		t.Fatal("expected cached.Metadata.Cached to be true")
	}
	if original.Metadata.Cached {
		t.Fatal("expected original.Metadata.Cached to remain false")
	}
	if len(cached.Results) != 1 || cached.Results[0].URL != "https://go.dev" {
		t.Fatalf("unexpected cached results: %+v", cached.Results)
	}

	// Mutate retrieved response to verify clone isolation
	cached.Results[0].Title = "Mutated Title"
	cached.Results[0].Content.Highlights[0] = "mutated highlight"

	secondGet, _ := c.Get(key)
	if secondGet.Results[0].Title == "Mutated Title" {
		t.Fatal("cache item was mutated by external modification; deep copy failed")
	}
	if secondGet.Results[0].Content.Highlights[0] == "mutated highlight" {
		t.Fatal("cache content slice was mutated; deep copy failed")
	}
}

func TestCache_TTLExpiration(t *testing.T) {
	shortTTL := 50 * time.Millisecond
	c := New(Options{TTL: shortTTL, MaxEntries: 10})

	key := "expire-key"
	resp := model.NewSearchResponse("test")
	c.Set(key, resp)

	// Immediate get succeeds
	if _, ok := c.Get(key); !ok {
		t.Fatal("expected item to be present immediately")
	}

	// Sleep past TTL
	time.Sleep(70 * time.Millisecond)

	// Post-expiration get fails and item is removed
	if _, ok := c.Get(key); ok {
		t.Fatal("expected item to expire and return not found")
	}
	if c.Len() != 0 {
		t.Fatalf("expected cache len=0 after expired get, got %d", c.Len())
	}
}

func TestCache_LRUEviction(t *testing.T) {
	maxEntries := 3
	c := New(Options{TTL: 10 * time.Minute, MaxEntries: maxEntries})

	for i := 1; i <= 3; i++ {
		c.Set(fmt.Sprintf("key-%d", i), model.NewSearchResponse(fmt.Sprintf("query-%d", i)))
	}
	if c.Len() != 3 {
		t.Fatalf("expected len 3, got %d", c.Len())
	}

	// Access key-1 to make it most recently used (MRU)
	if _, ok := c.Get("key-1"); !ok {
		t.Fatal("expected key-1 found")
	}

	// Insert key-4, which should evict key-2 (since key-1 was recently accessed, key-2 was oldest)
	c.Set("key-4", model.NewSearchResponse("query-4"))

	if c.Len() != 3 {
		t.Fatalf("expected len to remain at max 3, got %d", c.Len())
	}

	if _, ok := c.Get("key-2"); ok {
		t.Fatal("expected key-2 to have been evicted")
	}
	if _, ok := c.Get("key-1"); !ok {
		t.Fatal("expected key-1 to still be present (promoted by Get)")
	}
	if _, ok := c.Get("key-3"); !ok {
		t.Fatal("expected key-3 to be present")
	}
	if _, ok := c.Get("key-4"); !ok {
		t.Fatal("expected key-4 to be present")
	}
}

func TestCache_DisabledWhenTTLZero(t *testing.T) {
	c := New(Options{TTL: 0, MaxEntries: 100})
	key := "test"
	resp := model.NewSearchResponse("query")

	c.Set(key, resp)
	if c.Len() != 0 {
		t.Fatalf("expected len 0 when TTL=0, got %d", c.Len())
	}
	if _, ok := c.Get(key); ok {
		t.Fatal("expected get to return false when TTL=0")
	}
}

func TestCache_ConcurrentAccess(t *testing.T) {
	c := New(Options{TTL: 2 * time.Second, MaxEntries: 50})
	var wg sync.WaitGroup

	for i := 0; i < 20; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			for j := 0; j < 50; j++ {
				k := fmt.Sprintf("concurrent-key-%d", (id+j)%10)
				resp := model.NewSearchResponse(fmt.Sprintf("q-%d", j))
				c.Set(k, resp)
				c.Get(k)
				c.Len()
			}
		}(i)
	}

	wg.Wait()
	if c.Len() > 50 {
		t.Fatalf("expected len <= 50, got %d", c.Len())
	}
}
