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
}

// EngineSource records which engine discovered this result and its ranking position.
type EngineSource struct {
	Name     string `json:"name"`
	Position int    `json:"position"`
}

// ExtractedContent holds optional extracted text, markdown, or highlights.
type ExtractedContent struct {
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
