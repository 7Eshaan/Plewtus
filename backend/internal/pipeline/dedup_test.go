package pipeline

import (
	"strings"
	"testing"

	"plewtus/internal/model"
)

func TestDeduplicate_IdenticalURLsAcrossEngines(t *testing.T) {
	responses := []*model.ProviderResponse{
		{
			Engine: "duckduckgo",
			Results: []model.RawSearchResult{
				{
					Title:    "Go Programming",
					URL:      "https://go.dev/",
					Snippet:  "The Go Programming Language",
					Position: 1,
				},
				{
					Title:    "Learn Go",
					URL:      "https://learn.go.dev/",
					Snippet:  "Interactive tutorials",
					Position: 2,
				},
			},
		},
		{
			Engine: "bing",
			Results: []model.RawSearchResult{
				{
					Title:         "Go Programming Language",
					URL:           "https://www.go.dev/?utm_source=bing#home",
					Snippet:       "The Go Programming Language is open source.",
					Position:      1,
					PublishedDate: "2024-01-01",
				},
			},
		},
		{
			Engine: "yahoo",
			Results: []model.RawSearchResult{
				{
					Title:    "Go Programming",
					URL:      "http://go.dev/index.html",
					Snippet:  "Official Go website",
					Position: 3,
				},
			},
		},
	}

	results, dedupedCount, _ := DeduplicateDetailed(responses, DeduplicateOptions{Limit: 10})

	if dedupedCount != 2 {
		t.Fatalf("expected 2 unique pages, got %d", dedupedCount)
	}

	if len(results) != 2 {
		t.Fatalf("expected 2 results, got %d", len(results))
	}

	// First result should be go.dev (discovered by all 3 engines)
	first := results[0]
	if first.URL != "https://go.dev/" && first.URL != "https://www.go.dev/" {
		t.Errorf("expected go.dev URL, got %s", first.URL)
	}

	if len(first.Engines) != 3 {
		t.Fatalf("expected 3 discovering engines for go.dev, got %d: %+v", len(first.Engines), first.Engines)
	}

	// Verify all 3 engines recorded with their SERP positions
	engineMap := make(map[string]int)
	for _, e := range first.Engines {
		engineMap[e.Name] = e.Position
	}

	if pos, ok := engineMap["duckduckgo"]; !ok || pos != 1 {
		t.Errorf("expected duckduckgo position 1, got %d", pos)
	}
	if pos, ok := engineMap["bing"]; !ok || pos != 1 {
		t.Errorf("expected bing position 1, got %d", pos)
	}
	if pos, ok := engineMap["yahoo"]; !ok || pos != 3 {
		t.Errorf("expected yahoo position 3, got %d", pos)
	}

	// Verify PublishedDate was preserved from Bing
	if first.PublishedDate != "2024-01-01" {
		t.Errorf("expected published date '2024-01-01', got %q", first.PublishedDate)
	}

	// Verify snippet chosen was the richer one
	if first.Snippet != "The Go Programming Language is open source." {
		t.Errorf("expected richer snippet, got %q", first.Snippet)
	}
}

func TestDeduplicate_EngineConsensusRanking(t *testing.T) {
	responses := []*model.ProviderResponse{
		{
			Engine: "eng1",
			Results: []model.RawSearchResult{
				{Title: "Page Solo", URL: "https://solo.com", Position: 1},
				{Title: "Page Trio", URL: "https://trio.com", Position: 2},
				{Title: "Page Duo", URL: "https://duo.com", Position: 3},
			},
		},
		{
			Engine: "eng2",
			Results: []model.RawSearchResult{
				{Title: "Page Trio", URL: "https://trio.com", Position: 1},
				{Title: "Page Duo", URL: "https://duo.com", Position: 2},
			},
		},
		{
			Engine: "eng3",
			Results: []model.RawSearchResult{
				{Title: "Page Trio", URL: "https://trio.com", Position: 1},
			},
		},
	}

	results, dedupedCount, _ := DeduplicateDetailed(responses, DeduplicateOptions{Limit: 10})

	if dedupedCount != 3 {
		t.Fatalf("expected 3 unique pages, got %d", dedupedCount)
	}

	// Rank 1: Trio (3 engines)
	if results[0].Title != "Page Trio" {
		t.Errorf("rank 1 should be Trio (3 engines), got %s", results[0].Title)
	}
	if results[0].Rank != 1 {
		t.Errorf("expected rank 1, got %d", results[0].Rank)
	}

	// Rank 2: Duo (2 engines)
	if results[1].Title != "Page Duo" {
		t.Errorf("rank 2 should be Duo (2 engines), got %s", results[1].Title)
	}
	if results[1].Rank != 2 {
		t.Errorf("expected rank 2, got %d", results[1].Rank)
	}

	// Rank 3: Solo (1 engine)
	if results[2].Title != "Page Solo" {
		t.Errorf("rank 3 should be Solo (1 engine), got %s", results[2].Title)
	}
	if results[2].Rank != 3 {
		t.Errorf("expected rank 3, got %d", results[2].Rank)
	}
}

func TestDeduplicate_TitleTruncationSelection(t *testing.T) {
	responses := []*model.ProviderResponse{
		{
			Engine: "eng1",
			Results: []model.RawSearchResult{
				{
					Title:    "Go Concurrency Patte...",
					URL:      "https://example.com/concurrency",
					Position: 1,
				},
			},
		},
		{
			Engine: "eng2",
			Results: []model.RawSearchResult{
				{
					Title:    "Go Concurrency Patterns: Pipelines and Cancellation",
					URL:      "https://example.com/concurrency",
					Position: 1,
				},
			},
		},
	}

	results, _, _ := DeduplicateDetailed(responses, DeduplicateOptions{Limit: 10})
	if len(results) != 1 {
		t.Fatalf("expected 1 result, got %d", len(results))
	}

	if results[0].Title != "Go Concurrency Patterns: Pipelines and Cancellation" {
		t.Errorf("expected non-truncated title, got %q", results[0].Title)
	}
}

func TestDeduplicate_LimitEnforcement(t *testing.T) {
	responses := []*model.ProviderResponse{
		{
			Engine: "eng1",
			Results: []model.RawSearchResult{
				{Title: "1", URL: "https://1.com", Position: 1},
				{Title: "2", URL: "https://2.com", Position: 2},
				{Title: "3", URL: "https://3.com", Position: 3},
				{Title: "4", URL: "https://4.com", Position: 4},
				{Title: "5", URL: "https://5.com", Position: 5},
			},
		},
	}

	results, dedupedCount, _ := DeduplicateDetailed(responses, DeduplicateOptions{Limit: 2})

	if dedupedCount != 5 {
		t.Errorf("expected dedupedCount 5 before limit truncation, got %d", dedupedCount)
	}

	if len(results) != 2 {
		t.Errorf("expected 2 results due to limit, got %d", len(results))
	}
}

func TestDeduplicate_EmptyResponses(t *testing.T) {
	results, count, _ := DeduplicateDetailed(nil, DeduplicateOptions{Limit: 10})
	if len(results) != 0 || count != 0 {
		t.Errorf("expected empty results on nil input")
	}

	results, count, _ = DeduplicateDetailed([]*model.ProviderResponse{}, DeduplicateOptions{Limit: 10})
	if len(results) != 0 || count != 0 {
		t.Errorf("expected empty results on empty input")
	}
}

func TestDeduplicate_IncludeDomainsFilter(t *testing.T) {
	responses := []*model.ProviderResponse{
		{
			Engine: "engine",
			Results: []model.RawSearchResult{
				{Title: "Go Official", URL: "https://go.dev/doc", Position: 1},
				{Title: "Subdomain", URL: "https://pkg.go.dev/std", Position: 2},
				{Title: "Other Site", URL: "https://stackoverflow.com/questions", Position: 3},
				{Title: "GitHub", URL: "https://github.com/golang/go", Position: 4},
			},
		},
	}

	results, count, _ := DeduplicateDetailed(responses, DeduplicateOptions{
		Limit:          10,
		IncludeDomains: []string{"go.dev"},
	})

	if count != 2 {
		t.Fatalf("expected 2 results matching go.dev, got %d", count)
	}

	if len(results) != 2 {
		t.Fatalf("expected 2 results, got %d", len(results))
	}

	for _, r := range results {
		if r.URL != "https://go.dev/doc" && r.URL != "https://pkg.go.dev/std" {
			t.Errorf("unexpected URL outside include_domains: %s", r.URL)
		}
	}
}

func TestDeduplicate_ExcludeDomainsFilter(t *testing.T) {
	responses := []*model.ProviderResponse{
		{
			Engine: "engine",
			Results: []model.RawSearchResult{
				{Title: "Go Official", URL: "https://go.dev/doc", Position: 1},
				{Title: "Spam Site", URL: "https://spam.com/article", Position: 2},
				{Title: "Sub Spam", URL: "https://ads.spam.com/click", Position: 3},
				{Title: "Legit Site", URL: "https://example.com/clean", Position: 4},
			},
		},
	}

	results, count, _ := DeduplicateDetailed(responses, DeduplicateOptions{
		Limit:          10,
		ExcludeDomains: []string{"spam.com"},
	})

	if count != 2 {
		t.Fatalf("expected 2 results excluding spam.com, got %d", count)
	}

	for _, r := range results {
		if r.URL == "https://spam.com/article" || r.URL == "https://ads.spam.com/click" {
			t.Errorf("found excluded domain URL: %s", r.URL)
		}
	}
}

func TestDeduplicate_OffsetAndLimit(t *testing.T) {
	responses := []*model.ProviderResponse{
		{
			Engine: "engine",
			Results: []model.RawSearchResult{
				{Title: "P1", URL: "https://example.com/1", Position: 1},
				{Title: "P2", URL: "https://example.com/2", Position: 2},
				{Title: "P3", URL: "https://example.com/3", Position: 3},
				{Title: "P4", URL: "https://example.com/4", Position: 4},
				{Title: "P5", URL: "https://example.com/5", Position: 5},
			},
		},
	}

	// Offset 2, Limit 2 -> returns items at index 2 and 3 (P3 and P4)
	results, count, _ := DeduplicateDetailed(responses, DeduplicateOptions{
		Limit:  2,
		Offset: 2,
	})

	if count != 5 {
		t.Errorf("expected total count 5 before pagination, got %d", count)
	}

	if len(results) != 2 {
		t.Fatalf("expected 2 paginated results, got %d", len(results))
	}

	if results[0].Title != "P3" || results[1].Title != "P4" {
		t.Errorf("expected P3 and P4, got %s and %s", results[0].Title, results[1].Title)
	}

	// Offset beyond total -> empty slice
	beyondResults, _, _ := DeduplicateDetailed(responses, DeduplicateOptions{
		Limit:  2,
		Offset: 10,
	})
	if len(beyondResults) != 0 {
		t.Errorf("expected 0 results when offset exceeds total items, got %d", len(beyondResults))
	}
}

func TestDeduplicateDetailedReportsCandidateDecisions(t *testing.T) {
	responses := []*model.ProviderResponse{
		{Engine: "bing", Results: []model.RawSearchResult{
			{Title: "Go Docs", URL: "https://go.dev/doc", Snippet: "Go language documentation", Position: 1},
			{Title: "Go Docs", URL: "https://ads.example/doc", Snippet: "Go language documentation", Position: 2},
			{Title: "Go Docs", URL: "not a url ://", Snippet: "Go language documentation", Position: 3},
			{Title: "Unrelated sports", URL: "https://sports.example/", Snippet: "Football news", Position: 4},
		}},
		{Engine: "yahoo", Results: []model.RawSearchResult{
			{Title: "Official Go Docs", URL: "https://www.go.dev/doc?utm_source=test", Snippet: "Go language documentation", Position: 1},
		}},
	}
	results, deduped, stats := DeduplicateDetailed(responses, DeduplicateOptions{
		Query: "go docs", Limit: 10, ExcludeDomains: []string{"ads.example"},
	})
	if len(results) != 1 || deduped != 1 {
		t.Fatalf("expected the off-topic result to be dropped, got results=%d deduped=%d", len(results), deduped)
	}
	if !strings.Contains(results[0].Title, "Go Docs") {
		t.Fatalf("expected relevant result to be kept, got %q", results[0].Title)
	}
	if stats.RelevanceRejected != 1 || stats.DomainRejected != 1 || stats.InvalidURL != 1 || stats.Duplicates != 1 {
		t.Fatalf("unexpected candidate stats: %+v", stats)
	}
}

func TestDeduplicateMergesTitleAndSnippetByQueryFit(t *testing.T) {
	responses := []*model.ProviderResponse{
		{Engine: "bing", Results: []model.RawSearchResult{{
			Title: "Lando Norris", URL: "https://example.com/driver",
			Snippet: "A short profile of the McLaren Formula One driver.", Position: 1,
		}}},
		{Engine: "yahoo", Results: []model.RawSearchResult{{
			Title: "Formula One drivers", URL: "https://example.com/driver",
			Snippet: "Lando Norris is a British racing driver competing in Formula One.", Position: 2,
		}}},
	}
	results, _, _ := DeduplicateDetailed(responses, DeduplicateOptions{Query: "Lando Norris", Limit: 10})
	if len(results) != 1 {
		t.Fatalf("expected one merged result, got %d", len(results))
	}
	if results[0].Title != "Lando Norris" {
		t.Errorf("expected query-matching title to be retained, got %q", results[0].Title)
	}
	if !strings.Contains(results[0].Snippet, "Lando Norris") {
		t.Errorf("expected query-matching snippet to be retained, got %q", results[0].Snippet)
	}
}
