package model

import (
	"errors"
	"strings"
)

// ExtractOptions specifies options for webpage extraction.
type ExtractOptions struct {
	Enabled         bool   `json:"enabled"`
	Mode            string `json:"mode,omitempty"`               // "highlights", "text", "markdown"
	MaxPages        int    `json:"max_pages,omitempty"`          // default 3, max 10
	MaxCharsPerPage int    `json:"max_chars_per_page,omitempty"` // default 4000
}

// SearchRequest represents the public search API request body.
type SearchRequest struct {
	Query          string          `json:"query"`
	Limit          int             `json:"limit,omitempty"`
	Providers      []string        `json:"providers,omitempty"`
	Freshness      string          `json:"freshness,omitempty"` // "day", "week", "month", "year"
	IncludeDomains []string        `json:"include_domains,omitempty"`
	ExcludeDomains []string        `json:"exclude_domains,omitempty"`
	Country        string          `json:"country,omitempty"`
	Language       string          `json:"language,omitempty"`
	Extract        *ExtractOptions `json:"extract,omitempty"`
}

// Normalize cleans up and sets defaults for the request.
func (r *SearchRequest) Normalize(defaultLimit, maxLimit int, defaultProviders []string) {
	r.Query = strings.TrimSpace(r.Query)

	if r.Limit <= 0 {
		r.Limit = defaultLimit
	} else if r.Limit > maxLimit {
		r.Limit = maxLimit
	}

	if len(r.Providers) == 0 {
		r.Providers = append([]string(nil), defaultProviders...)
	} else {
		var cleaned []string
		seen := make(map[string]bool)
		for _, p := range r.Providers {
			low := strings.ToLower(strings.TrimSpace(p))
			if low != "" && !seen[low] {
				seen[low] = true
				cleaned = append(cleaned, low)
			}
		}
		if len(cleaned) == 0 {
			r.Providers = append([]string(nil), defaultProviders...)
		} else {
			r.Providers = cleaned
		}
	}

	r.Freshness = strings.ToLower(strings.TrimSpace(r.Freshness))
	r.Country = strings.ToLower(strings.TrimSpace(r.Country))
	r.Language = strings.ToLower(strings.TrimSpace(r.Language))

	var cleanInc []string
	for _, d := range r.IncludeDomains {
		trimmed := strings.ToLower(strings.TrimSpace(d))
		if trimmed != "" {
			cleanInc = append(cleanInc, trimmed)
		}
	}
	r.IncludeDomains = cleanInc

	var cleanExc []string
	for _, d := range r.ExcludeDomains {
		trimmed := strings.ToLower(strings.TrimSpace(d))
		if trimmed != "" {
			cleanExc = append(cleanExc, trimmed)
		}
	}
	r.ExcludeDomains = cleanExc

	if r.Extract != nil && r.Extract.Enabled {
		if r.Extract.Mode == "" {
			r.Extract.Mode = "highlights"
		} else {
			r.Extract.Mode = strings.ToLower(strings.TrimSpace(r.Extract.Mode))
		}
		if r.Extract.MaxPages <= 0 {
			r.Extract.MaxPages = 3
		} else if r.Extract.MaxPages > 10 {
			r.Extract.MaxPages = 10
		}
		if r.Extract.MaxCharsPerPage <= 0 {
			r.Extract.MaxCharsPerPage = 4000
		} else if r.Extract.MaxCharsPerPage > 20000 {
			r.Extract.MaxCharsPerPage = 20000
		}
	}
}

// Validate validates the incoming search request.
func (r *SearchRequest) Validate() error {
	if strings.TrimSpace(r.Query) == "" {
		return errors.New("query parameter is required and cannot be empty")
	}

	if r.Freshness != "" {
		switch r.Freshness {
		case "day", "week", "month", "year":
			// valid
		default:
			return errors.New("freshness must be one of: 'day', 'week', 'month', 'year'")
		}
	}

	if r.Extract != nil && r.Extract.Enabled {
		switch r.Extract.Mode {
		case "highlights", "text", "markdown", "metadata":
			// valid
		default:
			return errors.New("extract mode must be one of: 'highlights', 'text', 'markdown', 'metadata'")
		}
	}

	return nil
}
