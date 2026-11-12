package marginalia

import (
	"encoding/json"
	"strings"

	"plewtus/internal/model"
)

// apiResponse is the subset of the Marginalia API response used here.
type apiResponse struct {
	License string      `json:"license"`
	Query   string      `json:"query"`
	Results []apiResult `json:"results"`
}

type apiResult struct {
	URL         string `json:"url"`
	Title       string `json:"title"`
	Description string `json:"description"`
}

// ParseResponse decodes a Marginalia API response into raw results, skipping
// entries without an http(s) URL. Positions follow the API's ranking order.
func ParseResponse(body []byte) ([]model.RawSearchResult, error) {
	var parsed apiResponse
	if err := json.Unmarshal(body, &parsed); err != nil {
		return nil, err
	}

	results := make([]model.RawSearchResult, 0, len(parsed.Results))
	for _, r := range parsed.Results {
		link := strings.TrimSpace(r.URL)
		if !strings.HasPrefix(link, "http://") && !strings.HasPrefix(link, "https://") {
			continue
		}
		title := strings.Join(strings.Fields(r.Title), " ")
		if title == "" {
			title = link
		}
		results = append(results, model.RawSearchResult{
			Title:    title,
			URL:      link,
			Snippet:  strings.Join(strings.Fields(r.Description), " "),
			Position: len(results) + 1,
		})
	}
	return results, nil
}
