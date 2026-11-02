package pipeline

import (
	"sort"
	"strings"

	"plewtus/internal/model"
)

type mergedResult struct {
	canonicalKey  string
	url           string
	title         string
	snippet       string
	publishedDate string
	engines       []model.EngineSource
	bestPosition  int
	engineCount   int
	firstSeen     int
}

// Deduplicate takes raw provider responses from multiple search engines,
// canonicalizes and deduplicates URLs, merges engine positions, titles,
// and snippets, applies ranking (defaulting to RRFRanker if no custom ranker is passed),
// and returns the unified list of ranked results along with the total count
// of unique results before limit truncation.
func Deduplicate(responses []*model.ProviderResponse, limit int, rankers ...Ranker) (results []model.SearchResult, dedupedCount int) {
	if len(responses) == 0 {
		return []model.SearchResult{}, 0
	}

	// Sort provider responses by engine name for deterministic processing order
	sortedResponses := make([]*model.ProviderResponse, len(responses))
	copy(sortedResponses, responses)
	sort.Slice(sortedResponses, func(i, j int) bool {
		return sortedResponses[i].Engine < sortedResponses[j].Engine
	})

	seen := make(map[string]*mergedResult)
	var list []*mergedResult

	for _, resp := range sortedResponses {
		for _, raw := range resp.Results {
			if strings.TrimSpace(raw.URL) == "" {
				continue
			}

			cleanURL, key, err := NormalizeURL(raw.URL)
			if err != nil {
				cleanURL = strings.TrimSpace(raw.URL)
				key = strings.ToLower(cleanURL)
			}

			existing, found := seen[key]
			if !found {
				item := &mergedResult{
					canonicalKey:  key,
					url:           cleanURL,
					title:         strings.TrimSpace(raw.Title),
					snippet:       cleanSnippet(raw.Snippet),
					publishedDate: strings.TrimSpace(raw.PublishedDate),
					engines: []model.EngineSource{
						{
							Name:     resp.Engine,
							Position: raw.Position,
						},
					},
					bestPosition: raw.Position,
					engineCount:  1,
					firstSeen:    len(list),
				}
				seen[key] = item
				list = append(list, item)
			} else {
				// 1. Merge engine source and position
				engineRecorded := false
				for idx, eng := range existing.engines {
					if eng.Name == resp.Engine {
						engineRecorded = true
						if raw.Position < eng.Position {
							existing.engines[idx].Position = raw.Position
						}
						break
					}
				}
				if !engineRecorded {
					existing.engines = append(existing.engines, model.EngineSource{
						Name:     resp.Engine,
						Position: raw.Position,
					})
					existing.engineCount++
				}

				// 2. Update best position
				if raw.Position < existing.bestPosition {
					existing.bestPosition = raw.Position
				}

				// 3. Merge published date if not previously available
				if existing.publishedDate == "" && strings.TrimSpace(raw.PublishedDate) != "" {
					existing.publishedDate = strings.TrimSpace(raw.PublishedDate)
				}

				// 4. Select most informative title
				existing.title = selectBetterTitle(existing.title, raw.Title)

				// 5. Select most complete snippet
				existing.snippet = selectBetterSnippet(existing.snippet, raw.Snippet)

				// 6. Prefer secure HTTPS URL over insecure HTTP
				if strings.HasPrefix(cleanURL, "https://") && strings.HasPrefix(existing.url, "http://") {
					existing.url = cleanURL
				}
			}
		}
	}

	dedupedCount = len(list)
	if dedupedCount == 0 {
		return []model.SearchResult{}, 0
	}

	// Sort engine sources inside each result by SERP position
	for _, item := range list {
		sort.Slice(item.engines, func(i, j int) bool {
			if item.engines[i].Position != item.engines[j].Position {
				return item.engines[i].Position < item.engines[j].Position
			}
			return item.engines[i].Name < item.engines[j].Name
		})
	}

	// Build unranked SearchResult slice
	unranked := make([]model.SearchResult, len(list))
	for i, item := range list {
		unranked[i] = model.SearchResult{
			Title:         item.title,
			URL:           item.url,
			Snippet:       item.snippet,
			Engines:       item.engines,
			PublishedDate: item.publishedDate,
		}
	}

	// Apply Ranker (defaults to RRFRanker with k=60)
	var ranker Ranker = DefaultRanker
	if len(rankers) > 0 && rankers[0] != nil {
		ranker = rankers[0]
	}

	ranked := ranker.Rank(unranked)

	// Apply limit truncation
	if limit > 0 && len(ranked) > limit {
		ranked = ranked[:limit]
	}

	return ranked, dedupedCount
}

func selectBetterTitle(curr, candidate string) string {
	curr = strings.TrimSpace(curr)
	candidate = strings.TrimSpace(candidate)
	if curr == "" {
		return candidate
	}
	if candidate == "" {
		return curr
	}

	currTrunc := strings.HasSuffix(curr, "...") || strings.HasSuffix(curr, "…")
	candTrunc := strings.HasSuffix(candidate, "...") || strings.HasSuffix(candidate, "…")
	if currTrunc && !candTrunc {
		return candidate
	}
	if candTrunc && !currTrunc {
		return curr
	}

	if len(candidate) > len(curr) {
		return candidate
	}
	return curr
}

func selectBetterSnippet(curr, candidate string) string {
	curr = cleanSnippet(curr)
	candidate = cleanSnippet(candidate)
	if curr == "" {
		return candidate
	}
	if candidate == "" {
		return curr
	}

	currTrunc := strings.HasSuffix(curr, "...") || strings.HasSuffix(curr, "…")
	candTrunc := strings.HasSuffix(candidate, "...") || strings.HasSuffix(candidate, "…")
	if currTrunc && !candTrunc {
		return candidate
	}
	if candTrunc && !currTrunc {
		return curr
	}

	if len(candidate) > len(curr) {
		return candidate
	}
	return curr
}

func cleanSnippet(s string) string {
	s = strings.TrimSpace(s)
	s = strings.TrimSuffix(s, "Read more")
	s = strings.TrimSuffix(s, "… Read more")
	s = strings.TrimSuffix(s, "... Read more")
	return strings.TrimSpace(s)
}
