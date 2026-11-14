package all

import (
	"time"

	"plewtus/internal/provider"
	"plewtus/internal/provider/bing"
	"plewtus/internal/provider/brave"
	"plewtus/internal/provider/duckduckgo"
	"plewtus/internal/provider/marginalia"
	"plewtus/internal/provider/qwant"
	"plewtus/internal/provider/yahoo"
)

// Options carries provider-specific settings.
type Options struct {
	// MarginaliaAPIKey is sent to the Marginalia API; empty uses the shared public key.
	MarginaliaAPIKey string
	// BraveAPIKey switches Brave to its official Search API; empty uses the HTML page.
	BraveAPIKey string
}

// RegisterAll registers all available search providers into the specified registry.
func RegisterAll(reg *provider.Registry, timeout time.Duration, userAgent string, opts ...Options) {
	var o Options
	if len(opts) > 0 {
		o = opts[0]
	}
	reg.Register(duckduckgo.New(timeout, userAgent))
	reg.Register(bing.New(timeout, userAgent))
	reg.Register(yahoo.New(timeout, userAgent))
	reg.Register(marginalia.New(timeout, userAgent, o.MarginaliaAPIKey))
	reg.Register(brave.New(timeout, userAgent, o.BraveAPIKey))
	reg.Register(qwant.New(timeout, userAgent))
}

// NewRegistry creates a new Registry pre-configured with all active search providers.
func NewRegistry(timeout time.Duration, userAgent string, opts ...Options) *provider.Registry {
	reg := provider.NewRegistry()
	RegisterAll(reg, timeout, userAgent, opts...)
	return reg
}
