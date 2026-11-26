package pipeline

import (
	"fmt"
	"strings"
	"testing"

	"plewtus/internal/model"
)

// serp builds one engine's results from title/snippet pairs.
func serp(engine string, pairs ...[2]string) *model.ProviderResponse {
	resp := &model.ProviderResponse{Engine: engine}
	for i, p := range pairs {
		resp.Results = append(resp.Results, model.RawSearchResult{
			Title: p[0], Snippet: p[1], Position: i + 1,
			URL: fmt.Sprintf("https://%s.example/%d", engine, i),
		})
	}
	return resp
}

func hobbitPages(engine string) *model.ProviderResponse {
	return serp(engine,
		[2]string{"The Hobbit (novel by J.R.R. Tolkien) | Britannica", "The Hobbit, fantasy novel by J.R.R. Tolkien, published in 1937."},
		[2]string{"The Hobbit - Wikipedia", "The Hobbit, or There and Back Again is a children's fantasy novel by English author J. R. R. Tolkien."},
		[2]string{"Biography - The Tolkien Society", "Tolkien, author of The Hobbit and The Lord of the Rings, was born in 1892."},
		[2]string{"The Hobbit | Plot, Characters, & Facts", "Bilbo Baggins, the hobbit, joins Gandalf and thirteen dwarves."},
	)
}

// Natural-language questions carry words the answer pages need not repeat.
// With several engines agreeing on what the question is about ("hobbit"),
// pages missing the incidental word ("wrote") are relevant, and no engine is
// discarded as a decoy for it. (Benchmark finding: Bing's genuine results for
// this question were dropped.)
func TestPoolWeights_NaturalLanguageQuestionKeepsGenuineResults(t *testing.T) {
	// One engine also returns pages that repeat the question's wording; the
	// strict rule then treated them as the only relevant ones and discarded
	// the other engines' genuine pages.
	ddg := hobbitPages("duckduckgo")
	ddg.Results = append(ddg.Results,
		model.RawSearchResult{Title: "Who wrote The Hobbit? - Quora", Snippet: "J.R.R. Tolkien wrote The Hobbit, published in 1937.", URL: "https://quora.example/hobbit", Position: 5},
		model.RawSearchResult{Title: "Who Wrote The Hobbit | StatMuse Books", Snippet: "The Hobbit was written by J.R.R. Tolkien; he wrote it in the 1930s.", URL: "https://statmuse.example/hobbit", Position: 6},
	)
	_, available, stats := DeduplicateDetailed([]*model.ProviderResponse{
		hobbitPages("bing"), ddg, hobbitPages("yahoo"),
	}, DeduplicateOptions{Query: "who wrote the hobbit", Limit: 10})
	if len(stats.LowQualityEngines) != 0 {
		t.Fatalf("no engine should be discarded, got %v", stats.LowQualityEngines)
	}
	if stats.RelevanceRejected != 0 {
		t.Fatalf("genuine pages about The Hobbit were rejected: %d", stats.RelevanceRejected)
	}
	if available != 14 {
		t.Fatalf("expected all 14 distinct pages to be kept, got %d", available)
	}
}

// Decoy pages that match only a word the other engines' results do not use
// (as Bing served during the benchmark) are still discarded.
func TestPoolWeights_DictionaryDecoysStillDiscarded(t *testing.T) {
	var decoys [][2]string
	for i := 0; i < 10; i++ {
		decoys = append(decoys, [2]string{
			fmt.Sprintf("MEANING | English meaning - Cambridge Dictionary %d", i),
			"The meaning of a sentence often depends on stress and intonation.",
		})
	}
	harp := func(engine string) *model.ProviderResponse {
		return serp(engine,
			[2]string{"How the Harp Became the Symbol of Ireland", "Since the 13th century, the harp has been the heraldic symbol of Ireland."},
			[2]string{"The Harp of Ireland: A National Symbol", "The history of the Irish harp, from medieval courts to the coat of arms."},
			[2]string{"Irish harp - Wikipedia", "The harp is the national symbol of Ireland and appears on the arms of Ireland."},
			[2]string{"Why is the harp a symbol of Ireland?", "The harp appears on Irish coins and the presidential seal of Ireland."},
		)
	}
	results, _, stats := DeduplicateDetailed([]*model.ProviderResponse{
		serp("bing", decoys...), harp("duckduckgo"), harp("yahoo"),
	}, DeduplicateOptions{Query: "what is the meaning of the harp in ireland", Limit: 10})
	if len(stats.LowQualityEngines) != 1 || stats.LowQualityEngines[0] != "bing" {
		t.Fatalf("the decoy response must be discarded, got %v", stats.LowQualityEngines)
	}
	for _, r := range results {
		if strings.Contains(r.Title, "Dictionary") {
			t.Fatalf("decoy page returned: %s", r.Title)
		}
	}
}

// One engine cannot vouch for its own results: with a single engine the
// strict rule applies, so a decoy-only response is still discarded.
func TestPoolWeights_NotAppliedToASingleEngine(t *testing.T) {
	var decoys [][2]string
	for i := 0; i < 10; i++ {
		decoys = append(decoys, [2]string{
			fmt.Sprintf("MEANING Definition & Meaning - Merriam-Webster %d", i),
			"The meaning of MEANING is the thing one intends to convey.",
		})
	}
	results, _, stats := DeduplicateDetailed([]*model.ProviderResponse{serp("bing", decoys...)},
		DeduplicateOptions{Query: "what is the meaning of the harp in ireland", Limit: 10})
	if len(results) != 0 || len(stats.LowQualityEngines) != 1 {
		t.Fatalf("a lone decoy response must be discarded, got %d results, discarded %v", len(results), stats.LowQualityEngines)
	}
}

func TestPoolWeights_Weights(t *testing.T) {
	q := newQueryProfile("who wrote the hobbit").withPoolWeights([]string{
		"The Hobbit by Tolkien", "The Hobbit novel", "Tolkien wrote The Hobbit", "The Hobbit film",
	})
	if got := q.weights["hobbit"]; got != 1 {
		t.Fatalf("a term in every candidate weighs 1, got %v", got)
	}
	if got := q.weights[stem("wrote")]; got != poolWeightFloor+(1-poolWeightFloor)*0.25 {
		t.Fatalf("unexpected weight for 'wrote': %v", got)
	}
	if single := newQueryProfile("golang").withPoolWeights([]string{"go"}); single.weights != nil {
		t.Fatal("single-term queries need no weights")
	}
}

// Safety: decoys outnumbering real results in a two-word question (the case
// most exposed to pool weighting, since the decoy word then looks common)
// must still be filtered when another engine returned real results.
func TestPoolWeights_DecoyHeavyTwoTermQuestion(t *testing.T) {
	var decoys [][2]string
	for i := 0; i < 10; i++ {
		decoys = append(decoys, [2]string{
			fmt.Sprintf("WROTE Definition & Meaning - Merriam-Webster %d", i),
			"The meaning of WROTE is past tense of write.",
		})
	}
	real := serp("duckduckgo",
		[2]string{"The Hobbit - Wikipedia", "The Hobbit is a children's fantasy novel by J. R. R. Tolkien."},
		[2]string{"The Hobbit | Britannica", "The Hobbit, fantasy novel by J.R.R. Tolkien, published in 1937."},
		[2]string{"The Hobbit (1977 film)", "An animated adaptation of Tolkien's The Hobbit."},
		[2]string{"Who wrote The Hobbit?", "Tolkien wrote The Hobbit as a story for his children."},
	)
	results, _, stats := DeduplicateDetailed([]*model.ProviderResponse{serp("bing", decoys...), real},
		DeduplicateOptions{Query: "who wrote the hobbit", Limit: 10})
	for _, r := range results {
		if strings.Contains(r.Title, "Merriam-Webster") {
			t.Fatalf("decoy page returned (discarded engines %v): %s", stats.LowQualityEngines, r.Title)
		}
	}
}
