package orchestrator

import (
	"context"
	"fmt"
	"log/slog"
	"sort"
	"strings"
	"time"

	"plewtus/internal/cache"
	"plewtus/internal/extractor"
	"plewtus/internal/model"
	"plewtus/internal/pipeline"
	"plewtus/internal/provider"
	"plewtus/internal/reliability"
)

// Orchestrator coordinates concurrent search queries across multiple providers.
type Orchestrator struct {
	registry  *provider.Registry
	extractor extractor.ContentExtractor
	cache     cache.SearchCache
	breakers  *reliability.Manager
	timeout   time.Duration
	userAgent string
	logger    *slog.Logger
}

// New creates a new Orchestrator instance.
func New(registry *provider.Registry, timeout time.Duration, userAgent string, logger *slog.Logger) *Orchestrator {
	if timeout <= 0 {
		timeout = 6 * time.Second
	}
	if logger == nil {
		logger = slog.Default()
	}
	ext := extractor.New(extractor.ExtractorOptions{
		Timeout:   timeout,
		UserAgent: userAgent,
		Logger:    logger,
	})
	breakers := reliability.NewManager(reliability.BreakerOptions{
		FailureThreshold: 3,
		Cooldown:         30 * time.Second,
	})
	return &Orchestrator{
		registry:  registry,
		extractor: ext,
		breakers:  breakers,
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

// WithCache configures an in-memory search response cache.
func (o *Orchestrator) WithCache(c cache.SearchCache) *Orchestrator {
	o.cache = c
	return o
}

// WithBreakers configures a custom circuit breaker manager.
func (o *Orchestrator) WithBreakers(b *reliability.Manager) *Orchestrator {
	o.breakers = b
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

	// 1. Check in-memory cache if available
	var cacheKey string
	if o.cache != nil {
		cacheKey = o.cache.Key(req)
		if cachedResp, ok := o.cache.Get(cacheKey); ok {
			cachedResp.Metadata.TookMs = time.Since(start).Milliseconds()
			cachedResp.Metadata.Timestamp = time.Now().UTC().Format(time.RFC3339)
			o.logger.Debug("search response served from cache",
				slog.String("query", req.Query),
				slog.Int("results", len(cachedResp.Results)),
			)
			return cachedResp, nil
		}
	}

	// 2. Determine requested providers
	requested := o.resolveProviders(req.Providers)

	// 3. Filter providers through circuit breakers and determine active query set
	var providersToQuery []string
	var initialFailed []model.ProviderError

	for _, pName := range requested {
		if o.breakers != nil {
			breaker := o.breakers.Get(pName)
			if err := breaker.Allow(); err != nil {
				o.logger.Warn("provider circuit breaker open, skipping",
					slog.String("engine", pName),
					slog.Any("error", err),
				)
				initialFailed = append(initialFailed, model.ProviderError{
					Engine: pName,
					Error:  err.Error(),
				})
				continue
			}
		}
		providersToQuery = append(providersToQuery, pName)
	}

	// Provider Fallback: if all requested engines were tripped, attempt a healthy registered engine
	queried := append([]string(nil), providersToQuery...)
	if len(providersToQuery) == 0 && len(requested) > 0 {
		for _, candidate := range o.registry.Names() {
			if containsProvider(requested, candidate) {
				continue
			}
			if o.breakers == nil || o.breakers.Get(candidate).Allow() == nil {
				providersToQuery = append(providersToQuery, candidate)
				queried = append(queried, candidate)
				o.logger.Info("circuit breaker fallback: querying alternative healthy provider",
					slog.String("fallback_engine", candidate),
				)
				break
			}
		}
	}

	// 4. Dispatch queries to active providers concurrently
	ch := make(chan providerOutcome, len(providersToQuery))

	for _, pName := range providersToQuery {
		go func(name string) {
			defer func() {
				if r := recover(); r != nil {
					if o.breakers != nil {
						o.breakers.Get(name).ReportFailure(fmt.Errorf("panic: %v", r))
					}
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
				Offset:         0,
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
				if o.breakers != nil {
					o.breakers.Get(name).ReportFailure(err)
				}
				ch <- providerOutcome{
					name: name,
					err:  err,
				}
				return
			}
			if resp == nil {
				err := fmt.Errorf("provider %s returned an empty response", name)
				if o.breakers != nil {
					o.breakers.Get(name).ReportFailure(err)
				}
				ch <- providerOutcome{name: name, err: err}
				return
			}
			if len(resp.Results) == 0 {
				o.logger.Warn("provider returned no parseable search results",
					slog.String("engine", name),
					slog.Duration("took", resp.Took),
				)
			}

			if o.breakers != nil {
				o.breakers.Get(name).ReportSuccess()
			}
			ch <- providerOutcome{
				name:     name,
				response: resp,
			}
		}(pName)
	}

	// 5. Collect outcomes with context cancellation resilience
	var successful []string
	failed := append([]model.ProviderError(nil), initialFailed...)
	var rawResponses []*model.ProviderResponse
	rawCount := 0
	replied := make(map[string]bool)

collectLoop:
	for i := 0; i < len(providersToQuery); i++ {
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
			for _, pName := range providersToQuery {
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
	if len(successful) == 0 && len(failed) > 0 {
		failures := make([]string, 0, len(failed))
		for _, providerErr := range failed {
			failures = append(failures, providerErr.Engine+": "+providerErr.Error)
		}
		return nil, fmt.Errorf("all search providers failed: %s", strings.Join(failures, "; "))
	}

	// 6. Sort metadata slices for deterministic JSON output
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

	// 7. Canonicalize, deduplicate, filter domains, and merge results across successful providers
	dedupOpts := pipeline.DeduplicateOptions{
		Query:          req.Query,
		Limit:          req.Limit,
		Offset:         req.Offset,
		IncludeDomains: req.IncludeDomains,
		ExcludeDomains: req.ExcludeDomains,
	}
	uniqueResults, dedupedCount := pipeline.DeduplicateWithOptions(rawResponses, dedupOpts)
	if rawCount > 0 && len(uniqueResults) == 0 {
		o.logger.Warn("all raw provider results were rejected or filtered",
			slog.Int("raw_results", rawCount),
			slog.Bool("domain_filters_active", len(req.IncludeDomains) > 0 || len(req.ExcludeDomains) > 0),
		)
	}

	// 8. Optional AI-friendly content extraction
	if req.Extract != nil && req.Extract.Enabled && o.extractor != nil && len(uniqueResults) > 0 {
		extractOpts := *req.Extract
		extractOpts.Query = req.Query
		uniqueResults = o.extractor.ExtractBatch(ctx, uniqueResults, &extractOpts)
	}

	// 9. Build the final unified response
	resp := model.NewSearchResponse(req.Query)
	resp.Results = uniqueResults
	resp.Metadata.TotalResults = len(uniqueResults)
	resp.Metadata.TookMs = time.Since(start).Milliseconds()
	resp.Metadata.Providers.Queried = queried
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

	// 10. Cache the response for future queries if cache is enabled and we have valid results
	if o.cache != nil && cacheKey != "" && len(resp.Results) > 0 {
		o.cache.Set(cacheKey, resp)
	}

	return resp, nil
}

func containsProvider(providers []string, name string) bool {
	for _, provider := range providers {
		if provider == name {
			return true
		}
	}
	return false
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
