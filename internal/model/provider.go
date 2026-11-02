package model

import "time"

// ProviderRequest is the internal request passed to search provider scrapers.
type ProviderRequest struct {
	Query          string
	Limit          int
	Freshness      string
	Country        string
	Language       string
	IncludeDomains []string
	ExcludeDomains []string
	UserAgent      string
	Timeout        time.Duration
}

// ProviderResponse is the internal response returned by search provider scrapers.
type ProviderResponse struct {
	Engine  string
	Results []RawSearchResult
	Took    time.Duration
}

// RawSearchResult is an unranked, provider-specific result before normalization and deduplication.
type RawSearchResult struct {
	Title         string
	URL           string
	Snippet       string
	Position      int // 1-based original position in engine SERP
	PublishedDate string
}
