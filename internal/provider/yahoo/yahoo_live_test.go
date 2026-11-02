package yahoo

import (
	"context"
	"strings"
	"testing"
	"time"

	"plewtus/internal/model"
)

func TestYahoo_LiveQueries(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping live network test in short mode")
	}

	p := New(8*time.Second, "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/128.0.0.0 Safari/537.36")

	queries := []string{
		"golang concurrency",
		"apple stock news",
		"quantum computing",
	}

	for _, q := range queries {
		t.Run(q, func(t *testing.T) {
			ctx, cancel := context.WithTimeout(context.Background(), 8*time.Second)
			defer cancel()

			req := model.ProviderRequest{
				Query: q,
				Limit: 5,
			}

			resp, err := p.Search(ctx, req)
			if err != nil {
				t.Fatalf("live query %q failed: %v", q, err)
			}

			if resp.Engine != "yahoo" {
				t.Errorf("expected engine 'yahoo', got %q", resp.Engine)
			}

			if len(resp.Results) == 0 {
				t.Fatalf("expected at least 1 result for query %q, got 0", q)
			}

			t.Logf("Query %q returned %d results in %v", q, len(resp.Results), resp.Took)

			for i, r := range resp.Results {
				if r.Title == "" {
					t.Errorf("[%d] result title is empty", i)
				}
				if r.URL == "" || (!strings.HasPrefix(r.URL, "http://") && !strings.HasPrefix(r.URL, "https://")) {
					t.Errorf("[%d] invalid result URL: %q", i, r.URL)
				}
				if strings.Contains(r.URL, "r.search.yahoo.com") {
					t.Errorf("[%d] URL contains yahoo redirect tracker: %s", i, r.URL)
				}
				if r.Position != i+1 {
					t.Errorf("[%d] expected position %d, got %d", i, i+1, r.Position)
				}
				t.Logf("  [%d] %s -> %s (snippet: %.60s...)", r.Position, r.Title, r.URL, r.Snippet)
			}

			time.Sleep(1 * time.Second)
		})
	}
}
