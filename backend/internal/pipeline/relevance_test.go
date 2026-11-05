package pipeline

import (
	"testing"

	"plewtus/internal/model"
)

func TestQueryRelevanceRejectsUnrelatedSERPResults(t *testing.T) {
	query := "model context protocol stdio"
	for _, result := range []model.RawSearchResult{
		{Title: "Orlando City", Snippet: "All the latest soccer news."},
		{Title: "Bokep Viral", Snippet: "Streaming videos updated daily."},
	} {
		if score, relevant := QueryRelevance(query, result.Title, result.Snippet); relevant {
			t.Errorf("expected unrelated result %q to be rejected (score %.2f)", result.Title, score)
		}
	}
}

func TestQueryRelevanceAcceptsRelevantResults(t *testing.T) {
	tests := []struct {
		query, title, snippet string
	}{
		{
			query:   "model context protocol stdio",
			title:   "Model Context Protocol: Transports",
			snippet: "The stdio transport is used for local MCP servers.",
		},
		{
			query:   "Lando Norris",
			title:   "Lando Norris | McLaren Racing",
			snippet: "Read the latest news and updates about Lando Norris.",
		},
		{
			query:   "Rust TIOBE index",
			title:   "TIOBE Index for Rust",
			snippet: "The programming language popularity index.",
		},
	}
	for _, tc := range tests {
		if score, relevant := QueryRelevance(tc.query, tc.title, tc.snippet); !relevant {
			t.Errorf("expected relevant result %q for %q to pass (score %.2f)", tc.title, tc.query, score)
		}
	}
}

func TestDeduplicateWithOptionsFiltersBeforeRRF(t *testing.T) {
	responses := []*model.ProviderResponse{
		{
			Engine: "bing",
			Results: []model.RawSearchResult{
				{Title: "Orlando City", URL: "https://orlandocitysc.com/", Snippet: "Soccer news", Position: 1},
				{Title: "Model Context Protocol Transports", URL: "https://modelcontextprotocol.io/transports", Snippet: "The stdio transport", Position: 2},
			},
		},
		{
			Engine: "yahoo",
			Results: []model.RawSearchResult{
				{Title: "Bokep Viral", URL: "https://spam.example/", Snippet: "Streaming videos", Position: 1},
			},
		},
	}

	results, count := DeduplicateWithOptions(responses, DeduplicateOptions{
		Query: "model context protocol stdio",
		Limit: 10,
	})
	if count != 1 || len(results) != 1 {
		t.Fatalf("expected one relevant result after filtering, got count=%d results=%d", count, len(results))
	}
	if results[0].Title != "Model Context Protocol Transports" {
		t.Fatalf("unexpected result survived relevance filter: %q", results[0].Title)
	}
}

func TestRRFRankerUsesRelevanceToBreakEqualScores(t *testing.T) {
	results := DefaultRanker.Rank([]model.SearchResult{
		{Title: "Weak match", Relevance: 0.5, Engines: []model.EngineSource{{Name: "bing", Position: 1}}},
		{Title: "Strong match", Relevance: 1, Engines: []model.EngineSource{{Name: "yahoo", Position: 1}}},
	})
	if results[0].Title != "Strong match" {
		t.Fatalf("expected stronger lexical match first, got %q", results[0].Title)
	}
}
