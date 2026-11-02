package provider

import (
	"context"

	"plewtus/internal/model"
)

// Provider defines the interface that all search engine scrapers must implement.
type Provider interface {
	// Name returns the unique identifier of the search provider (e.g., "duckduckgo", "bing", "yahoo").
	Name() string

	// Search executes a search query against the provider's endpoint.
	Search(ctx context.Context, req model.ProviderRequest) (*model.ProviderResponse, error)
}
