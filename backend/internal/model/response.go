package model

import "time"

// SearchResponse is the top-level unified response returned to AI agents.
type SearchResponse struct {
	Query    string         `json:"query"`
	Results  []SearchResult `json:"results"`
	Metadata SearchMetadata `json:"metadata"`
}

// SearchResult represents a single deduplicated, query-relevance and consensus-ranked result.
type SearchResult struct {
	ID             string            `json:"id"`
	Title          string            `json:"title"`
	URL            string            `json:"url"`
	Snippet        string            `json:"snippet"`
	Rank           int               `json:"rank"`
	Score          float64           `json:"score"`
	RelevanceScore float64           `json:"relevance_score"`
	RankingScore   float64           `json:"ranking_score"`
	Relevance      float64           `json:"-"` // Internal query relevance input to the hybrid ranker.
	Engines        []EngineSource    `json:"engines"`
	PublishedDate  string            `json:"published_date,omitempty"`
	Content        *ExtractedContent `json:"content,omitempty"`
	// ContentError explains why extraction was requested for this result but
	// produced no content (blocked address, non-HTML page, timeout, ...).
	ContentError string `json:"content_error,omitempty"`
}

// EngineSource records which engine discovered this result and its ranking position.
type EngineSource struct {
	Name     string `json:"name"`
	Position int    `json:"position"`
}

// ExtractedContent holds optional extracted text, markdown, or highlights.
type ExtractedContent struct {
	Title         string   `json:"title,omitempty"`
	Text          string   `json:"text,omitempty"`
	Markdown      string   `json:"markdown,omitempty"`
	Highlights    []string `json:"highlights,omitempty"`
	Author        string   `json:"author,omitempty"`
	Description   string   `json:"description,omitempty"`
	ExtractedAt   string   `json:"extracted_at"`
	PublishedDate string   `json:"-"`
}

// SearchMetadata provides transparent details about the search execution.
type SearchMetadata struct {
	TotalResults int              `json:"total_results"`
	TookMs       int64            `json:"took_ms"`
	Providers    ProviderMetadata `json:"providers"`
	Stats        SearchStats      `json:"stats"`
	Filters      *AppliedFilters  `json:"filters,omitempty"`
	Cached       bool             `json:"cached,omitempty"`
	Timestamp    string           `json:"timestamp"`
}

// AppliedFilters echoes any active query constraints applied to the search.
type AppliedFilters struct {
	Freshness      string   `json:"freshness,omitempty"`
	IncludeDomains []string `json:"include_domains,omitempty"`
	ExcludeDomains []string `json:"exclude_domains,omitempty"`
	Country        string   `json:"country,omitempty"`
	Language       string   `json:"language,omitempty"`
	Offset         int      `json:"offset,omitempty"`
	Limit          int      `json:"limit,omitempty"`
}

// ProviderMetadata breaks down provider execution status.
type ProviderMetadata struct {
	Queried    []string         `json:"queried"`
	Successful []string         `json:"successful"`
	Failed     []ProviderError  `json:"failed"`
	TimingMs   map[string]int64 `json:"timing_ms"`
}

// ProviderError captures engine-level failures without failing the whole request.
type ProviderError struct {
	Engine string `json:"engine"`
	Error  string `json:"error"`
}

// SearchStats records result counts through the pipeline.
type SearchStats struct {
	RawCount               int `json:"raw_count"`
	DedupedCount           int `json:"deduped_count"`
	RelevanceRejectedCount int `json:"relevance_rejected_count"`
	DomainRejectedCount    int `json:"domain_rejected_count"`
	InvalidURLCount        int `json:"invalid_url_count"`
	DuplicateCount         int `json:"duplicate_count"`
	// ProviderDiscardedCount is the number of raw results dropped because their
	// whole provider response was discarded as off-topic.
	ProviderDiscardedCount int `json:"provider_discarded_count"`
}

// HealthResponse represents the response from GET /health.
type HealthResponse struct {
	Status        string  `json:"status"`
	Version       string  `json:"version"`
	UptimeSeconds float64 `json:"uptime_seconds"`
	Timestamp     string  `json:"timestamp"`
}

// ErrorResponse represents a standardized error payload.
type ErrorResponse struct {
	Error   string `json:"error"`
	Message string `json:"message"`
	Code    int    `json:"code"`
}

// NewSearchResponse creates a new SearchResponse with default empty slices for JSON consistency.
func NewSearchResponse(query string) *SearchResponse {
	return &SearchResponse{
		Query:   query,
		Results: []SearchResult{},
		Metadata: SearchMetadata{
			TotalResults: 0,
			TookMs:       0,
			Providers: ProviderMetadata{
				Queried:    []string{},
				Successful: []string{},
				Failed:     []ProviderError{},
				TimingMs:   map[string]int64{},
			},
			Stats: SearchStats{
				RawCount:     0,
				DedupedCount: 0,
			},
			Timestamp: time.Now().UTC().Format(time.RFC3339),
		},
	}
}

// Clone creates a deep copy of a SearchResponse so callers can mutate metadata without data races.
func (src *SearchResponse) Clone() *SearchResponse {
	if src == nil {
		return nil
	}

	dst := &SearchResponse{
		Query:    src.Query,
		Results:  make([]SearchResult, len(src.Results)), // non-nil even when empty
		Metadata: src.Metadata,
	}

	// Deep copy results slice and engine sources
	for i, r := range src.Results {
		dst.Results[i] = r
		if len(r.Engines) > 0 {
			dst.Results[i].Engines = make([]EngineSource, len(r.Engines))
			copy(dst.Results[i].Engines, r.Engines)
		}
		if r.Content != nil {
			cnt := *r.Content
			if len(r.Content.Highlights) > 0 {
				cnt.Highlights = make([]string, len(r.Content.Highlights))
				copy(cnt.Highlights, r.Content.Highlights)
			}
			dst.Results[i].Content = &cnt
		}
	}

	// Deep copy metadata slices. Keep empty slices non-nil so they serialize
	// as [] rather than null, matching the API contract.
	dst.Metadata.Providers.Queried = append([]string{}, src.Metadata.Providers.Queried...)
	dst.Metadata.Providers.Successful = append([]string{}, src.Metadata.Providers.Successful...)
	dst.Metadata.Providers.Failed = append([]ProviderError{}, src.Metadata.Providers.Failed...)
	if src.Metadata.Providers.TimingMs != nil {
		dst.Metadata.Providers.TimingMs = make(map[string]int64, len(src.Metadata.Providers.TimingMs))
		for name, tookMs := range src.Metadata.Providers.TimingMs {
			dst.Metadata.Providers.TimingMs[name] = tookMs
		}
	}

	if src.Metadata.Filters != nil {
		flt := *src.Metadata.Filters
		if src.Metadata.Filters.IncludeDomains != nil {
			flt.IncludeDomains = append([]string{}, src.Metadata.Filters.IncludeDomains...)
		}
		if src.Metadata.Filters.ExcludeDomains != nil {
			flt.ExcludeDomains = append([]string{}, src.Metadata.Filters.ExcludeDomains...)
		}
		dst.Metadata.Filters = &flt
	}

	return dst
}
