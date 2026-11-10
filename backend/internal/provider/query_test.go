package provider

import (
	"testing"

	"plewtus/internal/model"
)

func TestBuildSearchQuery(t *testing.T) {
	tests := []struct {
		name     string
		req      model.ProviderRequest
		expected string
	}{
		{
			name: "plain query",
			req: model.ProviderRequest{
				Query: "golang generics",
			},
			expected: "golang generics",
		},
		{
			name: "single include domain",
			req: model.ProviderRequest{
				Query:          "concurrency",
				IncludeDomains: []string{"go.dev"},
			},
			expected: "concurrency site:go.dev",
		},
		{
			name: "multiple include domains with boolean OR",
			req: model.ProviderRequest{
				Query:          "concurrency",
				IncludeDomains: []string{"go.dev", "golang.org"},
			},
			expected: "concurrency (site:go.dev OR site:golang.org)",
		},
		{
			name: "exclude domains",
			req: model.ProviderRequest{
				Query:          "concurrency",
				ExcludeDomains: []string{"spam.com", "ads.org"},
			},
			expected: "concurrency -site:spam.com -site:ads.org",
		},
		{
			name: "include and exclude domains together",
			req: model.ProviderRequest{
				Query:          "tutorial",
				IncludeDomains: []string{"go.dev", "learn-go.org"},
				ExcludeDomains: []string{"spam.com"},
			},
			expected: "tutorial (site:go.dev OR site:learn-go.org) -site:spam.com",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := BuildSearchQuery(tc.req)
			if got != tc.expected {
				t.Errorf("BuildSearchQuery() = %q, want %q", got, tc.expected)
			}
		})
	}
}

func TestAcceptLanguage(t *testing.T) {
	if got := AcceptLanguage(""); got != "en-US,en;q=0.9" {
		t.Fatalf("default Accept-Language = %q", got)
	}
	if got := AcceptLanguage(" FR "); got != "fr,en;q=0.8" {
		t.Fatalf("localized Accept-Language = %q", got)
	}
}

func TestNormalizeCountryCode(t *testing.T) {
	if got := NormalizeCountryCode("UK"); got != "gb" {
		t.Fatalf("UK alias normalized to %q, want gb", got)
	}
}

func TestAppendPageSkipsRepeatsAndContinuesPositions(t *testing.T) {
	seen := map[string]bool{}
	all, added := AppendPage(nil, seen, []model.RawSearchResult{{URL: "https://a", Position: 1}, {URL: "https://b", Position: 2}})
	if added != 2 {
		t.Fatalf("expected 2 new results, got %d", added)
	}
	all, added = AppendPage(all, seen, []model.RawSearchResult{{URL: "https://b", Position: 1}, {URL: "https://c", Position: 2}})
	if added != 1 || len(all) != 3 || all[2].URL != "https://c" || all[2].Position != 3 {
		t.Fatalf("unexpected merge: added=%d all=%+v", added, all)
	}
}
