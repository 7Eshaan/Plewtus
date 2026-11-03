package orchestrator

import (
	"context"
	"fmt"
	"log/slog"
	"sort"
	"time"

	"plewtus/internal/extractor"
	"plewtus/internal/model"
	"plewtus/internal/pipeline"
	"plewtus/internal/provider"
)

// Orchestrator coordinates concurrent search queries across multiple providers.
type Orchestrator struct {
	registry  *provider.Registry
	extractor extractor.ContentExtractor
	timeout   time.Duration
	userAgent string
	logger    *slog.Logger
}

// New creates a new Orchestrator instance.
func New(registry *provider.Registry, timeout time.Duration, userAgent string, logger *slog.Logger) *Orchestrator {
	if timeout <= 0 {
		timeout = 3 * time.Second
	}
	if logger == nil {
		logger = slog.Default()
	}
	ext := extractor.New(extractor.ExtractorOptions{
		Timeout:   timeout,
		UserAgent: userAgent,
		Logger:    logger,
	})
	return &Orchestrator{
		registry:  registry,
		extractor: ext,
		timeout:   timeout,
		userAgent: userAgent,
		logger:    logger,
	}
}

// WithExtractor allows replacing the default extractor with a custom one (useful for testing).
func (o *Orchestrator) WithExtractor(ext extractor.ContentExtractor) *Orchestrator {
	o.extractor = ext
	return o
}

// providerOutcome captures the response or error from a single provider execution.
type providerOutcome struct {
	name     string
	response *model.ProviderResponse
	err      error
}

// Search executes concurrent search queries across requested providers and fuses partial responses.
func (o *Orchestrator) Search(ctx context.Context, req *model.SearchRequest) (*model.SearchResponse, error) {
	if ctx.Err() != nil {
		return nil, ctx.Err()
	}

	start := time.Now()

	// 1. Determine providers to query
	providers := o.resolveProviders(req.Providers)

	// 2. Dispatch queries to each provider concurrently
	ch := make(chan providerOutcome, len(providers))

	for _, pName := range providers {
		go func(name string) {
			defer func() {
				if r := recover(); r != nil {
					ch <- providerOutcome{
						name: name,
						err:  fmt.Errorf("panic in provider %s: %v", name, r),
					}
				}
			}()

			p, ok := o.registry.Get(name)
			if !ok {
				ch <- providerOutcome{
					name: name,
					err:  fmt.Errorf("provider %q not found or not registered", name),
				}
				return
			}

			// Per-provider timeout bounded by parent context
			pCtx, cancel := context.WithTimeout(ctx, o.timeout)
			defer cancel()

			pReq := model.ProviderRequest{
				Query:          req.Query,
				Limit:          req.Limit + req.Offset,
				Offset:         req.Offset,
				Freshness:      req.Freshness,
				Country:        req.Country,
				Language:       req.Language,
				IncludeDomains: req.IncludeDomains,
				ExcludeDomains: req.ExcludeDomains,
				UserAgent:      o.userAgent,
				Timeout:        o.timeout,
			}

			resp, err := p.Search(pCtx, pReq)
			if err != nil {
				ch <- providerOutcome{
					name: name,
					err:  err,
				}
				return
			}

			ch <- providerOutcome{
				name:     name,
				response: resp,
			}
		}(pName)
	}

	// 3. Collect outcomes with context cancellation resilience
	var successful []string
	var failed []model.ProviderError
	var rawResponses []*model.ProviderResponse
	rawCount := 0
	replied := make(map[string]bool)

collectLoop:
	for i := 0; i < len(providers); i++ {
		select {
		case out := <-ch:
			replied[out.name] = true
			if out.err != nil {
				o.logger.Warn("provider search failed",
					slog.String("engine", out.name),
					slog.Any("error", out.err),
				)
				failed = append(failed, model.ProviderError{
					Engine: out.name,
					Error:  out.err.Error(),
				})
			} else {
				successful = append(successful, out.name)
				if out.response != nil {
					rawResponses = append(rawResponses, out.response)
					rawCount += len(out.response.Results)
				}
			}
		case <-ctx.Done():
			// Request context cancelled or timed out
			o.logger.Warn("search request context terminated before all providers completed",
				slog.Any("error", ctx.Err()),
			)

			// Drain any immediately available responses
			drainResponses(ch, replied, &successful, &failed, &rawResponses, &rawCount)

			// Mark remaining unreplied providers as failed due to context termination
			for _, pName := range providers {
				if !replied[pName] {
					failed = append(failed, model.ProviderError{
						Engine: pName,
						Error:  ctx.Err().Error(),
					})
				}
			}
			break collectLoop
		}
	}

	// If the parent context was cancelled and we have no successful results, return the context error
	if ctx.Err() != nil && len(successful) == 0 {
		return nil, ctx.Err()
	}

	// 4. Sort metadata slices for deterministic JSON output
	sort.Strings(successful)
	sort.Slice(failed, func(i, j int) bool {
		return failed[i].Engine < failed[j].Engine
	})

	if successful == nil {
		successful = []string{}
	}
	if failed == nil {
		failed = []model.ProviderError{}
	}

	// 5. Canonicalize, deduplicate, filter domains, and merge results across successful providers
	dedupOpts := pipeline.DeduplicateOptions{
		Limit:          req.Limit,
		Offset:         req.Offset,
		IncludeDomains: req.IncludeDomains,
		ExcludeDomains: req.ExcludeDomains,
	}
	uniqueResults, dedupedCount := pipeline.DeduplicateWithOptions(rawResponses, dedupOpts)

	// 6. Optional AI-friendly content extraction
	if req.Extract != nil && req.Extract.Enabled && o.extractor != nil && len(uniqueResults) > 0 {
		uniqueResults = o.extractor.ExtractBatch(ctx, uniqueResults, req.Extract)
	}

	// 7. Build the final unified response
	resp := model.NewSearchResponse(req.Query)
	resp.Results = uniqueResults
	resp.Metadata.TotalResults = len(uniqueResults)
	resp.Metadata.TookMs = time.Since(start).Milliseconds()
	resp.Metadata.Providers.Queried = providers
	resp.Metadata.Providers.Successful = successful
	resp.Metadata.Providers.Failed = failed
	resp.Metadata.Stats.RawCount = rawCount
	resp.Metadata.Stats.DedupedCount = dedupedCount

	// Populate applied filters in metadata if any filters were active
	if req.Freshness != "" || len(req.IncludeDomains) > 0 || len(req.ExcludeDomains) > 0 ||
		req.Country != "" || req.Language != "" || req.Offset > 0 {
		resp.Metadata.Filters = &model.AppliedFilters{
			Freshness:      req.Freshness,
			IncludeDomains: req.IncludeDomains,
			ExcludeDomains: req.ExcludeDomains,
			Country:        req.Country,
			Language:       req.Language,
			Offset:         req.Offset,
			Limit:          req.Limit,
		}
	}

	return resp, nil
}

// resolveProviders extracts a sanitized, unique list of providers to query.
func (o *Orchestrator) resolveProviders(requested []string) []string {
	if len(requested) == 0 {
		return o.registry.Names()
	}

	var providers []string
	seen := make(map[string]bool)
	for _, p := range requested {
		if !seen[p] && p != "" {
			seen[p] = true
			providers = append(providers, p)
		}
	}
	return providers
}

// drainResponses drains non-blocking outcomes from the channel.
func drainResponses(
	ch <-chan providerOutcome,
	replied map[string]bool,
	successful *[]string,
	failed *[]model.ProviderError,
	rawResponses *[]*model.ProviderResponse,
	rawCount *int,
) {
	for {
		select {
		case out := <-ch:
			replied[out.name] = true
			if out.err != nil {
				*failed = append(*failed, model.ProviderError{
					Engine: out.name,
					Error:  out.err.Error(),
				})
			} else {
				*successful = append(*successful, out.name)
				if out.response != nil {
					*rawResponses = append(*rawResponses, out.response)
					*rawCount += len(out.response.Results)
				}
			}
		default:
			return
		}
	}
}
