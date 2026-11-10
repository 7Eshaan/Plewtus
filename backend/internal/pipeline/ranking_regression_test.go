package pipeline

import (
	"testing"

	"plewtus/internal/model"
)

// Regression: a keyword-stuffed page found deep in one engine must not outrank
// a page that every engine ranks first.
func TestRanking_ConsensusBeatsKeywordStuffing(t *testing.T) {
	wiki := model.RawSearchResult{Title: "Raft (algorithm) - Wikipedia", URL: "https://en.wikipedia.org/wiki/Raft_(algorithm)", Snippet: "Raft is a consensus algorithm designed as an alternative to Paxos.", Position: 1}
	responses := []*model.ProviderResponse{
		{Engine: "bing", Results: []model.RawSearchResult{wiki}},
		{Engine: "duckduckgo", Results: []model.RawSearchResult{wiki}},
		{Engine: "yahoo", Results: []model.RawSearchResult{wiki, {
			Title: "raft consensus algorithm - best raft consensus algorithm deals", URL: "https://seo-spam.example/raft",
			Snippet: "raft consensus algorithm raft consensus algorithm", Position: 40,
		}}},
	}
	results, _, _ := DeduplicateDetailed(responses, DeduplicateOptions{Query: "raft consensus algorithm", Limit: 10})
	if len(results) != 2 || results[0].URL != "https://en.wikipedia.org/wiki/Raft_(algorithm)" {
		t.Fatalf("expected consensus result first, got %+v", results)
	}
	if results[0].RankingScore < results[1].RankingScore {
		t.Fatal("results must be ordered by ranking_score")
	}
}

// Regression: an off-topic (poisoned) provider SERP is discarded entirely when
// another provider returned on-topic results.
func TestDeduplicate_DiscardsPoisonedProviderResponse(t *testing.T) {
	junk := []model.RawSearchResult{}
	for i, title := range []string{"Global Fusion, Chembur | Zomato", "PayPoint Login", "Shiv Chalisa Lyrics", "Jeep Compass Review", "PayPoint Privacy"} {
		junk = append(junk, model.RawSearchResult{Title: title, URL: "https://junk.example/" + string(rune('a'+i)), Snippet: "unrelated", Position: i + 1})
	}
	junk = append(junk, model.RawSearchResult{Title: "Postgres VACUUM tuning guide", URL: "https://shared.example/vacuum", Snippet: "Tuning autovacuum in Postgres", Position: 6})
	responses := []*model.ProviderResponse{
		{Engine: "bing", Results: junk},
		{Engine: "duckduckgo", Results: []model.RawSearchResult{
			{Title: "Routine Vacuuming - PostgreSQL docs", URL: "https://www.postgresql.org/docs/current/routine-vacuuming.html", Snippet: "PostgreSQL vacuum tuning and autovacuum", Position: 1},
			{Title: "Postgres VACUUM tuning guide", URL: "https://shared.example/vacuum", Snippet: "Tuning autovacuum in Postgres", Position: 2},
		}},
	}
	results, _, stats := DeduplicateDetailed(responses, DeduplicateOptions{Query: "postgres vacuum tuning", Limit: 10})
	if len(stats.LowQualityEngines) != 1 || stats.LowQualityEngines[0] != "bing" {
		t.Fatalf("expected bing flagged as low quality, got %v", stats.LowQualityEngines)
	}
	for _, r := range results {
		for _, e := range r.Engines {
			if e.Name == "bing" {
				t.Fatalf("poisoned engine must not contribute results or consensus: %+v", r)
			}
		}
	}
	if len(results) != 2 {
		t.Fatalf("expected 2 on-topic results, got %d", len(results))
	}
}

// A short query whose results look off-topic lexically is not discarded when
// no other provider can vouch for it (it may be a typo or a semantic match).
func TestDeduplicate_PoisonGateKeepsShortQueriesWithoutAHealthyProvider(t *testing.T) {
	var rs []model.RawSearchResult
	for i := 0; i < 5; i++ {
		rs = append(rs, model.RawSearchResult{Title: "Inter Miami", URL: "https://x.example/" + string(rune('a'+i)), Snippet: "soccer", Position: i + 1})
	}
	results, _, stats := DeduplicateDetailed([]*model.ProviderResponse{{Engine: "bing", Results: rs}}, DeduplicateOptions{Query: "messi transfer", Limit: 10})
	if len(stats.LowQualityEngines) != 0 || len(results) != 5 {
		t.Fatalf("expected fallback to keep results, got %d results, low=%v", len(results), stats.LowQualityEngines)
	}
}

func TestRelevance_CJKQueries(t *testing.T) {
	if _, ok := QueryRelevance("東京 天気", "東京の天気予報", "今日の天気"); !ok {
		t.Fatal("expected CJK query to match text without word spaces")
	}
	if _, ok := QueryRelevance("東京 天気", "大阪のグルメ", "ラーメン"); ok {
		t.Fatal("expected unrelated CJK text to be rejected")
	}
}

// Regression: a lone provider answering a specific query with unrelated pages
// (Bing's decoy result pages) must not be passed through as results.
func TestDeduplicate_DiscardsUnrelatedLoneProviderResponse(t *testing.T) {
	var rs []model.RawSearchResult
	for i, title := range []string{"YouTube", "Welcome to Python.org", "Summer Place Apartment Homes", "Girls games - Play free online games", "PayPoint Login"} {
		rs = append(rs, model.RawSearchResult{Title: title, URL: "https://junk.example/" + string(rune('a'+i)), Snippet: "unrelated text", Position: i + 1})
	}
	results, deduped, stats := DeduplicateDetailed([]*model.ProviderResponse{{Engine: "bing", Results: rs}}, DeduplicateOptions{Query: "rust ownership borrow checker", Limit: 10})
	if len(results) != 0 || deduped != 0 {
		t.Fatalf("expected no results from an unrelated response, got %d", len(results))
	}
	if len(stats.LowQualityEngines) != 1 || stats.ProviderDiscarded != 5 {
		t.Fatalf("expected bing discarded with 5 results, got %+v", stats)
	}
}

// Regression: the engines' own ordering decides the ranking; a slightly better
// keyword match further down one engine's page must not jump to the top.
func TestRanking_EngineOrderBeatsKeywordOverlap(t *testing.T) {
	top := model.RawSearchResult{Title: "Hybrid Search Scoring (RRF) - Azure AI Search", URL: "https://learn.example/rrf", Snippet: "Reciprocal Rank Fusion merges ranked lists.", Position: 1}
	deep := model.RawSearchResult{Title: "Reciprocal Rank Fusion Algorithm explained", URL: "https://blog.example/rrf", Snippet: "The reciprocal rank fusion algorithm.", Position: 8}
	responses := []*model.ProviderResponse{
		{Engine: "duckduckgo", Results: []model.RawSearchResult{top, deep}},
		{Engine: "yahoo", Results: []model.RawSearchResult{top}},
	}
	results, _, _ := DeduplicateDetailed(responses, DeduplicateOptions{Query: "reciprocal rank fusion algorithm", Limit: 10})
	if len(results) != 2 || results[0].URL != top.URL {
		t.Fatalf("expected the result both engines rank first to stay first, got %+v", results)
	}
}

func TestRelevance_IgnoresSearchOperators(t *testing.T) {
	if _, ok := QueryRelevance("jaguar -car", "Jaguar - Wikipedia", "The jaguar is a large cat species."); !ok {
		t.Fatal("a negated term must not be required")
	}
	if _, ok := QueryRelevance("site:github.com rrf", "RRF implementation", "Reciprocal rank fusion (rrf) in Go"); !ok {
		t.Fatal("site: operators must not be required words")
	}
	if _, ok := QueryRelevance(`intitle:raft "consensus algorithm" -"paxos made simple"`, "Raft consensus algorithm", ""); !ok {
		t.Fatal("field operators and quotes must reduce to their content words")
	}
}

func TestRelevance_MatchesInflectedForms(t *testing.T) {
	cases := [][3]string{
		{"best laptops 2026", "Best Laptop Picks of 2026", ""},
		{"running shoes review", "Run Shoe Reviews", "We reviewed shoes for runners"},
		{"http caching strategies", "HTTP cache strategy guide", ""},
	}
	for _, c := range cases {
		if _, ok := QueryRelevance(c[0], c[1], c[2]); !ok {
			t.Errorf("expected %q to match %q", c[0], c[1])
		}
	}
	if _, ok := QueryRelevance("CAN bus", "Bus schedule and route map", "Public transit bus services."); ok {
		t.Error("stemming must not make unrelated text match")
	}
}
