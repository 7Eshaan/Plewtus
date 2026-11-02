package all

import (
	"time"

	"plewtus/internal/provider"
	"plewtus/internal/provider/bing"
	"plewtus/internal/provider/duckduckgo"
	"plewtus/internal/provider/yahoo"
)

// RegisterAll registers all available search providers into the specified registry.
func RegisterAll(reg *provider.Registry, timeout time.Duration, userAgent string) {
	reg.Register(duckduckgo.New(timeout, userAgent))
	reg.Register(bing.New(timeout, userAgent))
	reg.Register(yahoo.New(timeout, userAgent))
}

// NewRegistry creates a new Registry pre-configured with all active search providers.
func NewRegistry(timeout time.Duration, userAgent string) *provider.Registry {
	reg := provider.NewRegistry()
	RegisterAll(reg, timeout, userAgent)
	return reg
}
