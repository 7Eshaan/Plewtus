package model

import (
	"errors"
	"fmt"
	"strings"
	"unicode/utf8"
)

// MaxResultWindow is the deepest result position (offset + limit) the
// federated pipeline can serve; upstream engines return roughly one SERP page.
const MaxResultWindow = 50

// MaxQueryChars bounds the query length; longer text is not a search query
// and would be forwarded verbatim to every upstream engine.
const MaxQueryChars = 500

// ErrInvalidRequest marks caller errors (HTTP 400) detected after decoding,
// such as references to unknown providers.
var ErrInvalidRequest = errors.New("invalid request")

// InvalidRequestf builds an error that wraps ErrInvalidRequest.
func InvalidRequestf(format string, args ...any) error {
	return fmt.Errorf("%w: %s", ErrInvalidRequest, fmt.Sprintf(format, args...))
}

// ExtractOptions specifies options for webpage extraction.
type ExtractOptions struct {
	Enabled         bool   `json:"enabled"`
	Mode            string `json:"mode,omitempty"`               // "highlights", "text", "markdown", "metadata"
	MaxPages        int    `json:"max_pages,omitempty"`          // default 3, max 10
	MaxCharsPerPage int    `json:"max_chars_per_page,omitempty"` // default 4000
	Query           string `json:"-"`                            // Internal relevance hint for highlights
}

// SearchRequest represents the public search API request body.
type SearchRequest struct {
	Query          string          `json:"query"`
	Limit          int             `json:"limit,omitempty"`
	Offset         int             `json:"offset,omitempty"`
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

	if maxLimit <= 0 || maxLimit > MaxResultWindow {
		maxLimit = MaxResultWindow
	}
	if defaultLimit <= 0 || defaultLimit > maxLimit {
		defaultLimit = maxLimit
	}
	if r.Offset < 0 {
		r.Offset = 0
	}

	limitDefaulted := r.Limit <= 0
	if limitDefaulted {
		r.Limit = defaultLimit
	} else if r.Limit > maxLimit {
		r.Limit = maxLimit
	}
	// When the caller only paginates (offset without limit), shrink the default
	// page so it still fits inside the result window instead of failing.
	if limitDefaulted && r.Offset < MaxResultWindow && r.Offset+r.Limit > MaxResultWindow {
		r.Limit = MaxResultWindow - r.Offset
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
	seenInc := make(map[string]bool)
	for _, d := range r.IncludeDomains {
		cleaned := cleanDomainString(d)
		if cleaned != "" && !seenInc[cleaned] {
			seenInc[cleaned] = true
			cleanInc = append(cleanInc, cleaned)
		}
	}
	r.IncludeDomains = cleanInc

	var cleanExc []string
	seenExc := make(map[string]bool)
	for _, d := range r.ExcludeDomains {
		cleaned := cleanDomainString(d)
		if cleaned != "" && !seenExc[cleaned] {
			seenExc[cleaned] = true
			cleanExc = append(cleanExc, cleaned)
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

func cleanDomainString(d string) string {
	d = strings.ToLower(strings.TrimSpace(d))
	d = strings.TrimPrefix(d, "http://")
	d = strings.TrimPrefix(d, "https://")
	d = strings.TrimPrefix(d, "www.")
	d = strings.TrimPrefix(d, ".")
	d = strings.TrimRight(d, "/")
	if idx := strings.Index(d, "/"); idx != -1 {
		d = d[:idx]
	}
	if idx := strings.Index(d, ":"); idx != -1 {
		d = d[:idx]
	}
	return d
}

// Validate validates the incoming search request.
func (r *SearchRequest) Validate() error {
	if strings.TrimSpace(r.Query) == "" {
		return errors.New("query parameter is required and cannot be empty")
	}
	if utf8.RuneCountInString(r.Query) > MaxQueryChars {
		return fmt.Errorf("query cannot exceed %d characters", MaxQueryChars)
	}
	if r.Offset+r.Limit > MaxResultWindow {
		return fmt.Errorf("offset + limit cannot exceed the %d-result provider window; reduce either value", MaxResultWindow)
	}

	if r.Freshness != "" {
		switch r.Freshness {
		case "day", "week", "month", "year":
			// valid
		default:
			return errors.New("freshness must be one of: 'day', 'week', 'month', 'year'")
		}
	}

	if r.Country != "" {
		if len(r.Country) != 2 || !isAlphaString(r.Country) {
			return errors.New("country must be a 2-letter ISO 3166-1 alpha-2 code (e.g. 'us', 'gb', 'de')")
		}
	}

	if r.Language != "" {
		if len(r.Language) != 2 || !isAlphaString(r.Language) {
			return errors.New("language must be a 2-letter ISO 639-1 code (e.g. 'en', 'de', 'fr')")
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

func isAlphaString(s string) bool {
	for _, r := range s {
		if (r < 'a' || r > 'z') && (r < 'A' || r > 'Z') {
			return false
		}
	}
	return true
}
