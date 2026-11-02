package pipeline

import (
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

	results, dedupedCount := Deduplicate(responses, 10)

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

	results, dedupedCount := Deduplicate(responses, 10)

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

	results, _ := Deduplicate(responses, 10)
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

	results, dedupedCount := Deduplicate(responses, 2)

	if dedupedCount != 5 {
		t.Errorf("expected dedupedCount 5 before limit truncation, got %d", dedupedCount)
	}

	if len(results) != 2 {
		t.Errorf("expected 2 results due to limit, got %d", len(results))
	}
}

func TestDeduplicate_EmptyResponses(t *testing.T) {
	results, count := Deduplicate(nil, 10)
	if len(results) != 0 || count != 0 {
		t.Errorf("expected empty results on nil input")
	}

	results, count = Deduplicate([]*model.ProviderResponse{}, 10)
	if len(results) != 0 || count != 0 {
		t.Errorf("expected empty results on empty input")
	}
}
