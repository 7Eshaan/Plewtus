package pipeline

import (
	"fmt"
	"math"
	"sort"

	"plewtus/internal/model"
)

const (
	// DefaultRRFK is the standard smoothing parameter established by Cormack, Clarke, and Buettcher (SIGIR 2009).
	DefaultRRFK            = 60.0
	ConsensusRankingWeight = 0.20
	RelevanceRankingWeight = 0.80
)

// Ranker defines the interface for ranking and fusing search results.
// Decoupling ranking from result collection and deduplication allows swapping or chaining
// different ranking strategies, such as Reciprocal Rank Fusion (RRF), Borda count,
// or downstream neural/ML rerankers.
type Ranker interface {
	Rank(results []model.SearchResult) []model.SearchResult
}

// RRFRanker uses Reciprocal Rank Fusion (RRF) consensus and query relevance
// to produce the final hybrid result order. The public Score remains raw RRF.
// For each document d, its score is computed as:
//
//	score(d) = sum_{m in M} 1 / (k + r_m(d))
//
// where r_m(d) is the 1-based rank of document d in engine m,
// and k is the constant smoothing parameter (default 60.0).
type RRFRanker struct {
	k float64
}

// NewRRFRanker creates a new RRFRanker with the specified smoothing constant k.
// If k <= 0, DefaultRRFK (60.0) is used.
func NewRRFRanker(k ...float64) *RRFRanker {
	kVal := DefaultRRFK
	if len(k) > 0 && k[0] > 0 {
		kVal = k[0]
	}
	return &RRFRanker{k: kVal}
}

// DefaultRanker is the global default ranker using standard RRF (k=60.0).
var DefaultRanker = NewRRFRanker(DefaultRRFK)

// K returns the smoothing parameter used by this ranker.
func (r *RRFRanker) K() float64 {
	return r.k
}

// ComputeScore calculates the Reciprocal Rank Fusion score for a set of engine sources.
func (r *RRFRanker) ComputeScore(engines []model.EngineSource) float64 {
	if len(engines) == 0 {
		return 0.0
	}

	seenEngines := make(map[string]bool)
	var score float64

	for _, eng := range engines {
		if eng.Name == "" || seenEngines[eng.Name] {
			continue
		}
		seenEngines[eng.Name] = true

		pos := eng.Position
		if pos < 1 {
			pos = 1
		}

		score += 1.0 / (r.k + float64(pos))
	}

	return score
}

// Rank scores and re-orders results using query relevance and normalized RRF consensus.
func (r *RRFRanker) Rank(results []model.SearchResult) []model.SearchResult {
	if len(results) == 0 {
		return []model.SearchResult{}
	}

	type scoredItem struct {
		result       model.SearchResult
		rawScore     float64
		rankScore    float64
		relevance    float64
		bestPosition int
		engineCount  int
		originalIdx  int
	}

	// Normalize consensus against the theoretical rank-1 score for the engines
	// that contributed results. This lets relevance participate in the final
	// ordering without changing the externally reported RRF score.
	engineNames := make(map[string]struct{})
	for _, res := range results {
		for _, eng := range res.Engines {
			if eng.Name != "" {
				engineNames[eng.Name] = struct{}{}
			}
		}
	}
	maxConsensus := float64(len(engineNames)) / (r.k + 1)
	if maxConsensus == 0 {
		maxConsensus = 1
	}

	scored := make([]scoredItem, len(results))
	for i, res := range results {
		rawScore := r.ComputeScore(res.Engines)
		consensus := math.Min(1, rawScore/maxConsensus)
		relevance := math.Max(0, math.Min(1, res.Relevance))
		// Give query fit more weight than consensus. The relevance filter removes
		// obvious noise upstream; this blend lets strong agreement across engines
		// beat marginally better keyword overlap.
		rankScore := ConsensusRankingWeight*consensus + RelevanceRankingWeight*relevance

		bestPos := math.MaxInt32
		for _, eng := range res.Engines {
			if eng.Position > 0 && eng.Position < bestPos {
				bestPos = eng.Position
			}
		}
		if bestPos == math.MaxInt32 {
			bestPos = 1
		}

		scored[i] = scoredItem{
			result:       res,
			rawScore:     rawScore,
			rankScore:    rankScore,
			relevance:    relevance,
			bestPosition: bestPos,
			engineCount:  len(res.Engines),
			originalIdx:  i,
		}
	}

	// Order by the hybrid ranking score so that multi-engine consensus can
	// outweigh small lexical differences (keyword-stuffed pages found deep in a
	// single engine must not beat a page every engine ranks first). Ties fall
	// back to relevance, engine count, best SERP position, and discovery order.
	sort.SliceStable(scored, func(i, j int) bool {
		if math.Abs(scored[i].rankScore-scored[j].rankScore) > 1e-9 {
			return scored[i].rankScore > scored[j].rankScore
		}
		if math.Abs(scored[i].relevance-scored[j].relevance) > 1e-9 {
			return scored[i].relevance > scored[j].relevance
		}
		if scored[i].engineCount != scored[j].engineCount {
			return scored[i].engineCount > scored[j].engineCount
		}
		if scored[i].bestPosition != scored[j].bestPosition {
			return scored[i].bestPosition < scored[j].bestPosition
		}
		return scored[i].originalIdx < scored[j].originalIdx
	})

	ranked := make([]model.SearchResult, len(scored))
	for i, item := range scored {
		res := item.result
		rank := i + 1
		res.Rank = rank
		// Round to 6 decimal places for clean, transparent agent consumption
		res.Score = math.Round(item.rawScore*1e6) / 1e6
		res.RelevanceScore = math.Round(item.relevance*1e6) / 1e6
		res.RankingScore = math.Round(item.rankScore*1e6) / 1e6
		res.ID = fmt.Sprintf("res_%d", rank)
		ranked[i] = res
	}

	return ranked
}

// MultiRanker allows chaining multiple rankers sequentially (e.g. RRF followed by a secondary reranker).
type MultiRanker []Ranker

// Rank runs results through each ranker in sequence.
func (mr MultiRanker) Rank(results []model.SearchResult) []model.SearchResult {
	for _, r := range mr {
		if r != nil {
			results = r.Rank(results)
		}
	}
	return results
}
