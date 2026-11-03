package model

import (
	"encoding/json"
	"testing"
)

func TestSearchRequestNormalization(t *testing.T) {
	req := SearchRequest{
		Query:          "  golang scraper  ",
		Limit:          0,
		Offset:         -5,
		Providers:      []string{"DuckDuckGo", " Bing ", "duckduckgo"},
		Freshness:      " WEEK ",
		IncludeDomains: []string{"https://www.go.dev/", "HTTP://GOLANG.ORG:8080/path"},
		ExcludeDomains: []string{"https://spam.com/ads"},
		Country:        " US ",
		Language:       " EN ",
	}

	req.Normalize(10, 50, []string{"duckduckgo", "bing", "yahoo"})

	if req.Query != "golang scraper" {
		t.Errorf("expected trimmed query, got %q", req.Query)
	}
	if req.Limit != 10 {
		t.Errorf("expected default limit 10, got %d", req.Limit)
	}
	if req.Offset != 0 {
		t.Errorf("expected negative offset normalized to 0, got %d", req.Offset)
	}
	if len(req.Providers) != 2 {
		t.Fatalf("expected 2 unique providers, got %d", len(req.Providers))
	}
	if req.Providers[0] != "duckduckgo" || req.Providers[1] != "bing" {
		t.Errorf("unexpected providers: %v", req.Providers)
	}
	if req.Freshness != "week" {
		t.Errorf("expected lowercased freshness 'week', got %q", req.Freshness)
	}
	if req.Country != "us" {
		t.Errorf("expected lowercased country 'us', got %q", req.Country)
	}
	if req.Language != "en" {
		t.Errorf("expected lowercased language 'en', got %q", req.Language)
	}
	if len(req.IncludeDomains) != 2 || req.IncludeDomains[0] != "go.dev" || req.IncludeDomains[1] != "golang.org" {
		t.Errorf("expected cleaned include domains ['go.dev', 'golang.org'], got %v", req.IncludeDomains)
	}
	if len(req.ExcludeDomains) != 1 || req.ExcludeDomains[0] != "spam.com" {
		t.Errorf("expected cleaned exclude domains ['spam.com'], got %v", req.ExcludeDomains)
	}
}

func TestSearchRequestValidation(t *testing.T) {
	tests := []struct {
		name    string
		req     SearchRequest
		wantErr bool
	}{
		{
			name:    "empty query",
			req:     SearchRequest{Query: "   "},
			wantErr: true,
		},
		{
			name:    "valid query",
			req:     SearchRequest{Query: "golang"},
			wantErr: false,
		},
		{
			name:    "valid freshness year",
			req:     SearchRequest{Query: "golang", Freshness: "year"},
			wantErr: false,
		},
		{
			name:    "invalid freshness",
			req:     SearchRequest{Query: "golang", Freshness: "decade"},
			wantErr: true,
		},
		{
			name:    "valid country",
			req:     SearchRequest{Query: "golang", Country: "us"},
			wantErr: false,
		},
		{
			name:    "invalid country 3 letters",
			req:     SearchRequest{Query: "golang", Country: "usa"},
			wantErr: true,
		},
		{
			name:    "invalid country digits",
			req:     SearchRequest{Query: "golang", Country: "12"},
			wantErr: true,
		},
		{
			name:    "valid language",
			req:     SearchRequest{Query: "golang", Language: "en"},
			wantErr: false,
		},
		{
			name:    "invalid language 3 letters",
			req:     SearchRequest{Query: "golang", Language: "eng"},
			wantErr: true,
		},
		{
			name: "invalid extract mode",
			req: SearchRequest{
				Query: "golang",
				Extract: &ExtractOptions{
					Enabled: true,
					Mode:    "audio",
				},
			},
			wantErr: true,
		},
		{
			name: "valid extract mode markdown",
			req: SearchRequest{
				Query: "golang",
				Extract: &ExtractOptions{
					Enabled: true,
					Mode:    "markdown",
				},
			},
			wantErr: false,
		},
		{
			name: "valid extract mode metadata",
			req: SearchRequest{
				Query: "golang",
				Extract: &ExtractOptions{
					Enabled: true,
					Mode:    "metadata",
				},
			},
			wantErr: false,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			err := tc.req.Validate()
			if (err != nil) != tc.wantErr {
				t.Errorf("Validate() error = %v, wantErr = %v", err, tc.wantErr)
			}
		})
	}
}

func TestNewSearchResponseJSON(t *testing.T) {
	resp := NewSearchResponse("test query")
	data, err := json.Marshal(resp)
	if err != nil {
		t.Fatalf("failed to marshal SearchResponse: %v", err)
	}

	var parsed SearchResponse
	if err := json.Unmarshal(data, &parsed); err != nil {
		t.Fatalf("failed to unmarshal SearchResponse: %v", err)
	}

	if parsed.Query != "test query" {
		t.Errorf("expected query 'test query', got %q", parsed.Query)
	}
	if parsed.Results == nil {
		t.Errorf("expected results to be non-nil empty slice")
	}
	if parsed.Metadata.Providers.Queried == nil {
		t.Errorf("expected providers.queried to be non-nil empty slice")
	}
}
