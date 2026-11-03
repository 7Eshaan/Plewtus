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
