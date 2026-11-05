package provider

import (
	"strings"

	"plewtus/internal/model"
)

// BuildSearchQuery constructs a clean search query containing search terms,
// domain inclusion filters (grouped with Boolean OR if multiple), and domain exclusion filters.
func BuildSearchQuery(req model.ProviderRequest) string {
	var parts []string
	if trimmed := strings.TrimSpace(req.Query); trimmed != "" {
		parts = append(parts, trimmed)
	}

	if len(req.IncludeDomains) == 1 {
		parts = append(parts, "site:"+req.IncludeDomains[0])
	} else if len(req.IncludeDomains) > 1 {
		var orParts []string
		for _, inc := range req.IncludeDomains {
			orParts = append(orParts, "site:"+inc)
		}
		parts = append(parts, "("+strings.Join(orParts, " OR ")+")")
	}

	for _, exc := range req.ExcludeDomains {
		parts = append(parts, "-site:"+exc)
	}

	return strings.Join(parts, " ")
}

// AcceptLanguage builds a consistent preference header for provider requests.
// Search engines still receive their own provider-specific language parameter;
// this header keeps redirects and localized interstitials aligned with it.
func AcceptLanguage(language string) string {
	language = strings.ToLower(strings.TrimSpace(language))
	if language == "" {
		return "en-US,en;q=0.9"
	}
	return language + ",en;q=0.8"
}

// NormalizeCountryCode converts commonly used aliases to ISO alpha-2 values.
func NormalizeCountryCode(country string) string {
	country = strings.ToLower(strings.TrimSpace(country))
	if country == "uk" {
		return "gb"
	}
	return country
}
