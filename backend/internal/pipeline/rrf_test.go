package pipeline

import (
	"math"
	"testing"

	"plewtus/internal/model"
)

func TestRRFRanker_Formula(t *testing.T) {
	ranker := NewRRFRanker(60.0)

	// Document discovered at position 1 on two engines:
	// score = 1/(60+1) + 1/(60+1) = 2/61 ≈ 0.032786885
	engines := []model.EngineSource{
		{Name: "bing", Position: 1},
		{Name: "yahoo", Position: 1},
	}

	score := ranker.ComputeScore(engines)
	expected := (1.0 / 61.0) + (1.0 / 61.0)

	if math.Abs(score-expected) > 1e-9 {
		t.Errorf("expected score %f, got %f", expected, score)
	}

	rounded := math.Round(score*1e6) / 1e6
	if rounded != 0.032787 {
		t.Errorf("expected rounded score 0.032787, got %f", rounded)
	}
}

func TestRRFRanker_MultiEngineConsensusBeatsSingleHighRank(t *testing.T) {
	ranker := DefaultRanker

	// Doc A: Rank 1 on 1 engine -> 1/61 ≈ 0.016393
	// Doc B: Rank 2 on 2 engines -> 1/62 + 1/62 = 2/62 ≈ 0.032258
	// Doc C: Rank 1 on 2 engines -> 1/61 + 1/61 = 2/61 ≈ 0.032787
	// Doc D: Rank 3 on 3 engines -> 1/63 + 1/63 + 1/63 = 3/63 ≈ 0.047619
	docs := []model.SearchResult{
		{
			ID:    "doc_a",
			Title: "Doc A (Solo Rank 1)",
			URL:   "https://a.com",
			Engines: []model.EngineSource{
				{Name: "bing", Position: 1},
			},
		},
		{
			ID:    "doc_b",
			Title: "Doc B (Duo Rank 2)",
			URL:   "https://b.com",
			Engines: []model.EngineSource{
				{Name: "bing", Position: 2},
				{Name: "yahoo", Position: 2},
			},
		},
		{
			ID:    "doc_c",
			Title: "Doc C (Duo Rank 1)",
			URL:   "https://c.com",
			Engines: []model.EngineSource{
				{Name: "bing", Position: 1},
				{Name: "yahoo", Position: 1},
			},
		},
		{
			ID:    "doc_d",
			Title: "Doc D (Trio Rank 3)",
			URL:   "https://d.com",
			Engines: []model.EngineSource{
				{Name: "bing", Position: 3},
				{Name: "yahoo", Position: 3},
				{Name: "duckduckgo", Position: 3},
			},
		},
	}

	ranked := ranker.Rank(docs)

	// Expected ranking:
	// 1. Doc D (score ~0.047619, 3 engines)
	// 2. Doc C (score ~0.032787, 2 engines)
	// 3. Doc B (score ~0.032258, 2 engines)
	// 4. Doc A (score ~0.016393, 1 engine)
	expectedOrder := []string{"Doc D (Trio Rank 3)", "Doc C (Duo Rank 1)", "Doc B (Duo Rank 2)", "Doc A (Solo Rank 1)"}

	for i, expectedTitle := range expectedOrder {
		if ranked[i].Title != expectedTitle {
			t.Errorf("rank %d: expected %q, got %q (score=%f)", i+1, expectedTitle, ranked[i].Title, ranked[i].Score)
		}
		if ranked[i].Rank != i+1 {
			t.Errorf("rank %d: expected rank field %d, got %d", i+1, i+1, ranked[i].Rank)
		}
		if ranked[i].ID != "res_"+string(rune('1'+i)) {
			t.Errorf("rank %d: expected ID res_%d, got %s", i+1, i+1, ranked[i].ID)
		}
	}
}

func TestRRFRanker_TieBreaking(t *testing.T) {
	ranker := DefaultRanker

	// Two docs with identical scores: both rank 1 on 1 engine
	docs := []model.SearchResult{
		{
			Title: "First Discovered",
			URL:   "https://first.com",
			Engines: []model.EngineSource{
				{Name: "bing", Position: 1},
			},
		},
		{
			Title: "Second Discovered",
			URL:   "https://second.com",
			Engines: []model.EngineSource{
				{Name: "yahoo", Position: 1},
			},
		},
	}

	ranked := ranker.Rank(docs)

	if len(ranked) != 2 {
		t.Fatalf("expected 2 results, got %d", len(ranked))
	}

	if ranked[0].Title != "First Discovered" {
		t.Errorf("expected first discovered to retain rank 1 on tie, got %s", ranked[0].Title)
	}
	if ranked[1].Title != "Second Discovered" {
		t.Errorf("expected second discovered to retain rank 2 on tie, got %s", ranked[1].Title)
	}

	if ranked[0].Rank != 1 || ranked[1].Rank != 2 {
		t.Errorf("expected ranks 1 and 2, got %d and %d", ranked[0].Rank, ranked[1].Rank)
	}
}

func TestRRFRanker_CustomK(t *testing.T) {
	ranker := NewRRFRanker(10.0)
	score := ranker.ComputeScore([]model.EngineSource{
		{Name: "bing", Position: 1},
	})
	expected := 1.0 / 11.0 // 1 / (10 + 1)
	if math.Abs(score-expected) > 1e-9 {
		t.Errorf("expected score %f, got %f", expected, score)
	}
}

func TestRRFRanker_EdgeCases(t *testing.T) {
	ranker := DefaultRanker

	// Empty input
	empty := ranker.Rank(nil)
	if len(empty) != 0 {
		t.Errorf("expected empty slice on nil input")
	}

	// Doc with empty engines
	noEngines := ranker.Rank([]model.SearchResult{
		{Title: "No Engines", URL: "https://none.com"},
	})
	if len(noEngines) != 1 || noEngines[0].Score != 0.0 {
		t.Errorf("expected score 0 for doc without engines, got %f", noEngines[0].Score)
	}

	// Doc with non-positive position
	nonPositivePos := ranker.ComputeScore([]model.EngineSource{
		{Name: "bing", Position: 0},
	})
	expected := 1.0 / 61.0 // clamped to 1
	if math.Abs(nonPositivePos-expected) > 1e-9 {
		t.Errorf("expected clamped score %f, got %f", expected, nonPositivePos)
	}

	// Doc with duplicate engine name in sources
	dupEngines := ranker.ComputeScore([]model.EngineSource{
		{Name: "bing", Position: 1},
		{Name: "bing", Position: 5},
	})
	if math.Abs(dupEngines-expected) > 1e-9 {
		t.Errorf("expected duplicate engine to be ignored, got %f", dupEngines)
	}
}
