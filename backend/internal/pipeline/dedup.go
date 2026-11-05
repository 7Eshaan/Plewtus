package pipeline

import (
	"net/url"
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
	relevance     float64
}

// DeduplicateOptions configures the result aggregation, filtering, ranking, and pagination.
type DeduplicateOptions struct {
	Query          string
	Limit          int
	Offset         int
	IncludeDomains []string
	ExcludeDomains []string
	Ranker         Ranker
}

// Deduplicate takes raw provider responses from multiple search engines,
// canonicalizes and deduplicates URLs, merges engine positions, titles,
// and snippets, applies ranking (defaulting to RRFRanker if no custom ranker is passed),
// and returns the unified list of ranked results along with the total count
// of unique results before limit truncation.
func Deduplicate(responses []*model.ProviderResponse, limit int, rankers ...Ranker) (results []model.SearchResult, dedupedCount int) {
	var ranker Ranker
	if len(rankers) > 0 {
		ranker = rankers[0]
	}
	return DeduplicateWithOptions(responses, DeduplicateOptions{
		Limit:  limit,
		Ranker: ranker,
	})
}

// DeduplicateWithOptions performs deduplication, domain inclusion/exclusion filtering,
// RRF ranking, and offset/limit pagination over raw provider responses.
func DeduplicateWithOptions(responses []*model.ProviderResponse, opts DeduplicateOptions) (results []model.SearchResult, dedupedCount int) {
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
			relevance, relevant := QueryRelevance(opts.Query, raw.Title, raw.Snippet)
			if !relevant {
				continue
			}

			cleanURL, key, err := NormalizeURL(raw.URL)
			if err != nil {
				cleanURL = strings.TrimSpace(raw.URL)
				key = strings.ToLower(cleanURL)
			}

			// Domain filtering: verify against include and exclude constraints
			if !matchesDomainFilters(cleanURL, opts.IncludeDomains, opts.ExcludeDomains) {
				continue
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
					relevance:    relevance,
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
				if relevance > existing.relevance {
					existing.relevance = relevance
				}

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
			Relevance:     item.relevance,
			Engines:       item.engines,
			PublishedDate: item.publishedDate,
		}
	}

	// Apply Ranker (defaults to RRFRanker with k=60)
	var ranker Ranker = DefaultRanker
	if opts.Ranker != nil {
		ranker = opts.Ranker
	}

	ranked := ranker.Rank(unranked)

	// Apply pagination (offset & limit)
	if opts.Offset > 0 {
		if opts.Offset < len(ranked) {
			ranked = ranked[opts.Offset:]
		} else {
			ranked = []model.SearchResult{}
		}
	}

	if opts.Limit > 0 && len(ranked) > opts.Limit {
		ranked = ranked[:opts.Limit]
	}

	return ranked, dedupedCount
}

func matchesDomainFilters(rawURL string, includeDomains, excludeDomains []string) bool {
	if len(includeDomains) == 0 && len(excludeDomains) == 0 {
		return true
	}

	parsed, err := url.Parse(rawURL)
	if err != nil || parsed.Hostname() == "" {
		return false
	}

	host := strings.ToLower(parsed.Hostname())
	host = strings.TrimSuffix(host, ".")

	// 1. Exclude check: reject if host matches or is a subdomain of any excluded domain
	for _, exc := range excludeDomains {
		exc = strings.TrimPrefix(strings.ToLower(strings.TrimSpace(exc)), "www.")
		exc = strings.TrimPrefix(exc, ".")
		if exc != "" && (host == exc || strings.HasSuffix(host, "."+exc)) {
			return false
		}
	}

	// 2. Include check: if specified, host must match or be a subdomain of at least one included domain
	if len(includeDomains) > 0 {
		matched := false
		for _, inc := range includeDomains {
			inc = strings.TrimPrefix(strings.ToLower(strings.TrimSpace(inc)), "www.")
			inc = strings.TrimPrefix(inc, ".")
			if inc != "" && (host == inc || strings.HasSuffix(host, "."+inc)) {
				matched = true
				break
			}
		}
		if !matched {
			return false
		}
	}

	return true
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
