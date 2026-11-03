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
