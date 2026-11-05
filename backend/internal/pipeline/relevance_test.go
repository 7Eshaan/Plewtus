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

func TestQueryRelevanceAllowsPartialOverlapForParaphrasedResults(t *testing.T) {
	score, relevant := QueryRelevance(
		"rust language adoption curve",
		"Rust Adoption Is Growing Fast",
		"A look at the programming language's growing ecosystem.",
	)
	if !relevant || score <= 0 {
		t.Fatalf("expected a relevant partial-overlap result to pass, got relevant=%v score=%.2f", relevant, score)
	}
}

func TestQueryRelevanceRejectsLowCoverageForLongQueries(t *testing.T) {
	query := "model context protocol stdio transport specification"
	score, relevant := QueryRelevance(query, "Model Protocol Overview", "A general guide to protocols.")
	if relevant {
		t.Fatalf("expected low-coverage result to be rejected, score %.2f", score)
	}
}

func TestQueryRelevanceAcceptsExactPhrase(t *testing.T) {
	query := "model context protocol stdio"
	score, relevant := QueryRelevance(query, "Model Context Protocol stdio transport", "")
	if !relevant || score < 0.9 {
		t.Fatalf("expected exact title phrase to be a strong match, relevant=%v score=%.2f", relevant, score)
	}
}

func TestQueryRelevancePreservesNegationAndProgrammingLanguageTokens(t *testing.T) {
	partialScore, _ := QueryRelevance("no internet connection", "Internet connection available", "Connect to the internet.")
	exactScore, exactRelevant := QueryRelevance("no internet connection", "No internet connection: troubleshooting guide", "")
	if !exactRelevant || partialScore >= exactScore {
		t.Fatalf("expected the exact negated topic to score above a partial overlap: partial=%.2f exact=%.2f", partialScore, exactScore)
	}
	if _, relevant := QueryRelevance("C++ memory safety", "C memory safety overview", "Memory safety in the C language."); relevant {
		t.Fatal("expected C++ query to reject results about C")
	}
	if _, relevant := QueryRelevance("C++ memory safety", "C++ memory safety", "Safer systems programming."); !relevant {
		t.Fatal("expected exact C++ result to pass")
	}
	if _, relevant := QueryRelevance("CAN bus", "Bus schedule and route map", "Public transit bus services."); relevant {
		t.Fatal("expected a query for CAN bus to reject generic transit-bus results")
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

func TestRRFRankerLetsStrongQueryFitBeatWeakEngineConsensus(t *testing.T) {
	results := DefaultRanker.Rank([]model.SearchResult{
		{
			Title:     "Weak consensus match",
			Relevance: 0.60,
			Engines: []model.EngineSource{
				{Name: "bing", Position: 2},
				{Name: "yahoo", Position: 2},
			},
		},
		{
			Title:     "Strong exact match",
			Relevance: 1,
			Engines:   []model.EngineSource{{Name: "bing", Position: 1}},
		},
	})
	if results[0].Title != "Strong exact match" {
		t.Fatalf("expected query fit to beat weak consensus, got %q", results[0].Title)
	}
	if results[0].Score >= results[1].Score {
		t.Fatal("expected exposed score to retain pure RRF values even when hybrid ranking order differs")
	}
}
