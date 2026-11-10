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
	relevant      bool
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

// DeduplicationStats records candidate rejection and merge reasons.
type DeduplicationStats struct {
	RelevanceRejected int
	DomainRejected    int
	InvalidURL        int
	Duplicates        int
	// ProviderDiscarded counts raw results dropped because their whole provider
	// response was discarded (see LowQualityEngines).
	ProviderDiscarded int
	// LowQualityEngines lists providers whose whole response was discarded
	// because almost none of it matched the query (anti-bot result poisoning).
	LowQualityEngines []string
}

const (
	// A provider response needs at least this many candidates before the
	// result-poisoning gate is applied, so tiny legitimate responses survive.
	lowQualityMinCandidates = 4
	// Responses where fewer than this fraction of candidates match the query
	// are treated as poisoned/off-topic SERPs.
	lowQualityRelevantFraction = 0.25
	// A low-quality response is discarded even when no other provider can vouch
	// for the query if its results cover, on average, less than this fraction of
	// the query terms and the query is specific enough (lowQualityMinTerms) for
	// that to be meaningful. Short queries stay untouched: engines legitimately
	// answer typos and one- or two-word queries with lexically different pages.
	lowQualityMeanCoverage = 0.34
	lowQualityMinTerms     = 3
)

type candidate struct {
	raw       model.RawSearchResult
	cleanURL  string
	key       string
	relevance float64
	relevant  bool
}

// DeduplicateDetailed canonicalizes and deduplicates raw provider responses,
// applies domain and relevance filtering, discards poisoned provider
// responses, ranks (RRFRanker unless opts.Ranker is set), and paginates. It
// returns the page of results, the number of unique results before pagination,
// and why raw results were rejected or merged.
func DeduplicateDetailed(responses []*model.ProviderResponse, opts DeduplicateOptions) (results []model.SearchResult, dedupedCount int, stats DeduplicationStats) {
	if len(responses) == 0 {
		return []model.SearchResult{}, 0, stats
	}

	// Sort provider responses by engine name for deterministic processing order
	sortedResponses := make([]*model.ProviderResponse, 0, len(responses))
	for _, resp := range responses {
		if resp != nil {
			sortedResponses = append(sortedResponses, resp)
		}
	}
	sort.Slice(sortedResponses, func(i, j int) bool {
		return sortedResponses[i].Engine < sortedResponses[j].Engine
	})

	// 1. Validate, normalize, and score every raw result per provider.
	query := newQueryProfile(opts.Query)
	perResponse := make([][]candidate, len(sortedResponses))
	lowQuality := make([]bool, len(sortedResponses))
	meanCoverage := make([]float64, len(sortedResponses))
	healthyResponses := 0
	for i, resp := range sortedResponses {
		relevantCount := 0
		coverageSum := 0.0
		for _, raw := range resp.Results {
			if strings.TrimSpace(raw.URL) == "" {
				stats.InvalidURL++
				continue
			}
			cleanURL, key, err := NormalizeURL(raw.URL)
			if err != nil {
				stats.InvalidURL++
				continue
			}
			if !matchesDomainFilters(cleanURL, opts.IncludeDomains, opts.ExcludeDomains) {
				stats.DomainRejected++
				continue
			}
			relevance, relevant, coverage := query.match(raw.Title, raw.Snippet)
			if relevant {
				relevantCount++
			}
			coverageSum += coverage
			perResponse[i] = append(perResponse[i], candidate{raw: raw, cleanURL: cleanURL, key: key, relevance: relevance, relevant: relevant})
		}
		n := len(perResponse[i])
		if n > 0 {
			meanCoverage[i] = coverageSum / float64(n)
		}
		if query.hasTerms() && n >= lowQualityMinCandidates && float64(relevantCount)/float64(n) < lowQualityRelevantFraction {
			lowQuality[i] = true
		} else if relevantCount > 0 {
			healthyResponses++
		}
	}

	// 2. Discard poisoned provider responses: when another provider produced
	// on-topic results, or when the response barely touches a specific query at
	// all. Otherwise the query is simply hard to match lexically (typos, short
	// queries, other scripts) and the response is kept.
	seen := make(map[string]*mergedResult)
	var list []*mergedResult
	for i, resp := range sortedResponses {
		unrelated := len(query.terms) >= lowQualityMinTerms && meanCoverage[i] < lowQualityMeanCoverage
		if lowQuality[i] && (healthyResponses > 0 || unrelated) {
			stats.LowQualityEngines = append(stats.LowQualityEngines, resp.Engine)
			stats.ProviderDiscarded += len(perResponse[i])
			continue
		}
		for _, c := range perResponse[i] {
			raw := c.raw
			existing, found := seen[c.key]
			if !found {
				item := &mergedResult{
					canonicalKey:  c.key,
					url:           c.cleanURL,
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
					relevance:    c.relevance,
					relevant:     c.relevant,
				}
				seen[c.key] = item
				list = append(list, item)
				continue
			}

			stats.Duplicates++
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

			// 4. Merge the best title and snippet independently. A strong snippet
			// must not cause a weaker title from the same provider response to
			// replace a better title from another engine.
			existing.title = query.selectTitle(existing.title, strings.TrimSpace(raw.Title))
			existing.snippet = query.selectSnippet(existing.snippet, cleanSnippet(raw.Snippet))
			if candRel, candRelevant := query.relevance(existing.title, existing.snippet); candRel > existing.relevance {
				existing.relevance = candRel
				existing.relevant = existing.relevant || candRelevant
			}
			existing.relevant = existing.relevant || c.relevant

			// 5. Prefer secure HTTPS URL over insecure HTTP
			if strings.HasPrefix(c.cleanURL, "https://") && strings.HasPrefix(existing.url, "http://") {
				existing.url = c.cleanURL
			}
		}
	}

	// 3. Drop results that do not match the query. If nothing matches at all,
	// keep the candidates rather than returning an empty page.
	relevantList := list[:0:0]
	for _, item := range list {
		if item.relevant {
			relevantList = append(relevantList, item)
		}
	}
	if len(relevantList) > 0 {
		stats.RelevanceRejected = len(list) - len(relevantList)
		list = relevantList
	}

	dedupedCount = len(list)
	if dedupedCount == 0 {
		return []model.SearchResult{}, 0, stats
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

	return ranked, dedupedCount, stats
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

// preferComplete picks the more complete of two texts: a non-truncated one
// over a truncated one, otherwise the longer.
func preferComplete(curr, candidate string) string {
	if curr == "" {
		return candidate
	}
	if candidate == "" {
		return curr
	}
	currTrunc := strings.HasSuffix(curr, "...") || strings.HasSuffix(curr, "…")
	candTrunc := strings.HasSuffix(candidate, "...") || strings.HasSuffix(candidate, "…")
	if currTrunc != candTrunc {
		if currTrunc {
			return candidate
		}
		return curr
	}
	if len(candidate) > len(curr) {
		return candidate
	}
	return curr
}

func selectBetterTitle(curr, candidate string) string {
	return preferComplete(strings.TrimSpace(curr), strings.TrimSpace(candidate))
}

func selectBetterSnippet(curr, candidate string) string {
	return preferComplete(cleanSnippet(curr), cleanSnippet(candidate))
}

func cleanSnippet(s string) string {
	s = strings.TrimSpace(s)
	s = strings.TrimSpace(strings.TrimSuffix(s, "Read more"))
	return s
}
