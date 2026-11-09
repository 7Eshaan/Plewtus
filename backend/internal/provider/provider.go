package provider

import (
	"context"
	"sort"
	"sync"

	"plewtus/internal/model"
)

// Provider defines the interface that all search engine scrapers must implement.
type Provider interface {
	// Name returns the unique identifier of the search provider (e.g. "duckduckgo", "bing", "yahoo").
	Name() string

	// Search executes a search query against the provider's endpoint.
	Search(ctx context.Context, req model.ProviderRequest) (*model.ProviderResponse, error)
}

// Registry manages the set of available search providers.
type Registry struct {
	mu        sync.RWMutex
	providers map[string]Provider
}

// NewRegistry creates a new, empty Registry.
func NewRegistry() *Registry {
	return &Registry{
		providers: make(map[string]Provider),
	}
}

// Register registers a provider. If a provider with the same name exists, it is overwritten.
func (r *Registry) Register(p Provider) {
	if p == nil {
		return
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	r.providers[p.Name()] = p
}

// Get returns the provider by name, if registered.
func (r *Registry) Get(name string) (Provider, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	p, ok := r.providers[name]
	return p, ok
}

// All returns all registered providers sorted by name.
func (r *Registry) All() []Provider {
	r.mu.RLock()
	defer r.mu.RUnlock()
	list := make([]Provider, 0, len(r.providers))
	for _, p := range r.providers {
		list = append(list, p)
	}
	sort.Slice(list, func(i, j int) bool {
		return list[i].Name() < list[j].Name()
	})
	return list
}

// Names returns a sorted list of all registered provider names.
func (r *Registry) Names() []string {
	r.mu.RLock()
	defer r.mu.RUnlock()
	names := make([]string, 0, len(r.providers))
	for name := range r.providers {
		names = append(names, name)
	}
	sort.Strings(names)
	return names
}
