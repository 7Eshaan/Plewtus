package orchestrator

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"sort"
	"strings"
	"sync"
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
	// extractTimeout bounds the content-extraction stage separately from the
	// provider stage so slow engines cannot starve page extraction.
	extractTimeout time.Duration
	// stragglerGrace, when positive, is how long the last outstanding provider
	// is given once every other provider has returned results (three or more
	// providers only). It bounds tail latency when one engine hangs.
	stragglerGrace time.Duration
	userAgent      string
	logger         *slog.Logger
	flightMu       sync.Mutex
	flights        map[string]*searchFlight
}

type searchFlight struct {
	done     chan struct{}
	response *model.SearchResponse
	err      error
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
		registry:       registry,
		extractor:      ext,
		breakers:       breakers,
		timeout:        timeout,
		extractTimeout: timeout,
		userAgent:      userAgent,
		logger:         logger,
		flights:        make(map[string]*searchFlight),
	}
}

// WithExtractTimeout sets the time budget for the extraction stage.
func (o *Orchestrator) WithExtractTimeout(d time.Duration) *Orchestrator {
	if d > 0 {
		o.extractTimeout = d
	}
	return o
}

// WithStragglerGrace sets how long the last outstanding provider may take
// after all the others have returned results. Zero disables the cut-off.
func (o *Orchestrator) WithStragglerGrace(d time.Duration) *Orchestrator {
	if d >= 0 {
		o.stragglerGrace = d
	}
	return o
}

// ProviderNames returns the registered provider names.
func (o *Orchestrator) ProviderNames() []string {
	return o.registry.Names()
}

// overallTimeout caps a whole search: provider stage, optional extraction
// stage, and a small allowance for ranking and serialization.
func (o *Orchestrator) overallTimeout(req *model.SearchRequest) time.Duration {
	d := o.timeout + time.Second
	if req.Extract != nil && req.Extract.Enabled {
		d += o.extractTimeout
	}
	return d
}

// validateProviders rejects provider names that are not registered, before
// any work (or circuit-breaker allocation) happens for them.
func (o *Orchestrator) validateProviders(req *model.SearchRequest) error {
	var unknown []string
	for _, name := range req.Providers {
		if _, ok := o.registry.Get(name); !ok {
			unknown = append(unknown, name)
		}
	}
	if len(unknown) > 0 {
		return model.InvalidRequestf("unknown provider(s) %s; available: %s",
			strings.Join(unknown, ", "), strings.Join(o.registry.Names(), ", "))
	}
	return nil
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
	tookMs   int64
}

// Search executes concurrent search queries across requested providers and fuses partial responses.
func (o *Orchestrator) Search(ctx context.Context, req *model.SearchRequest) (response *model.SearchResponse, err error) {
	start := time.Now()
	if req == nil {
		return nil, fmt.Errorf("search request cannot be nil")
	}
	if ctx.Err() != nil {
		return nil, ctx.Err()
	}
	if err := o.validateProviders(req); err != nil {
		return nil, err
	}
	if o.cache == nil {
		searchCtx, cancel := context.WithTimeout(ctx, o.overallTimeout(req))
		defer cancel()
		return o.search(searchCtx, req, "")
	}

	cacheKey := o.cache.Key(req)
	if cachedResp, ok := o.cache.Get(cacheKey); ok {
		cachedResp.Metadata.TookMs = time.Since(start).Milliseconds()
		cachedResp.Metadata.Timestamp = time.Now().UTC().Format(time.RFC3339)
		return cachedResp, nil
	}

	o.flightMu.Lock()
	flight, exists := o.flights[cacheKey]
	if !exists {
		flight = &searchFlight{done: make(chan struct{})}
		o.flights[cacheKey] = flight

		// Execute the search flight in a goroutine bounded by the provider timeout,
		// detached from individual caller cancellation so an aborted client doesn't kill
		// inflight work for other concurrent callers.
		go func() {
			var resp *model.SearchResponse
			var searchErr error
			defer func() {
				if recovered := recover(); recovered != nil {
					searchErr = fmt.Errorf("search orchestration panic: %v", recovered)
					o.logger.Error("search orchestration panic recovered", slog.Any("error", recovered))
				}
				o.flightMu.Lock()
				flight.response = resp
				flight.err = searchErr
				delete(o.flights, cacheKey)
				close(flight.done)
				o.flightMu.Unlock()
			}()

			flightCtx, cancel := context.WithTimeout(context.Background(), o.overallTimeout(req))
			defer cancel()
			resp, searchErr = o.search(flightCtx, req, cacheKey)
		}()
	}
	o.flightMu.Unlock()

	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	case <-flight.done:
		if flight.err != nil {
			return nil, flight.err
		}
		cloned := flight.response.Clone()
		if cloned != nil {
			cloned.Metadata.TookMs = time.Since(start).Milliseconds()
			cloned.Metadata.Timestamp = time.Now().UTC().Format(time.RFC3339)
		}
		return cloned, nil
	}
}

func (o *Orchestrator) search(ctx context.Context, req *model.SearchRequest, cacheKey string) (*model.SearchResponse, error) {
	if ctx.Err() != nil {
		return nil, ctx.Err()
	}

	start := time.Now()

	// 1. Determine requested providers
	requested := o.resolveProviders(req.Providers)

	// 2. Filter providers through circuit breakers and determine active query set
	var providersToQuery []string
	var initialFailed []model.ProviderError
	breakerGenerations := make(map[string]uint64, len(requested))

	for _, pName := range requested {
		if o.breakers != nil {
			breaker := o.breakers.Get(pName)
			generation, err := breaker.AllowWithGeneration()
			if err != nil {
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
			breakerGenerations[pName] = generation
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
			if o.breakers == nil {
				providersToQuery = append(providersToQuery, candidate)
				queried = append(queried, candidate)
				break
			}
			generation, err := o.breakers.Get(candidate).AllowWithGeneration()
			if err == nil {
				breakerGenerations[candidate] = generation
				providersToQuery = append(providersToQuery, candidate)
				queried = append(queried, candidate)
				o.logger.Info("circuit breaker fallback: querying alternative healthy provider",
					slog.String("fallback_engine", candidate),
				)
				break
			}
		}
	}

	// 4. Dispatch queries to active providers concurrently. provCtx lets the
	// collector cancel a straggler without cancelling the whole search.
	provCtx, cancelProviders := context.WithCancel(ctx)
	defer cancelProviders()

	ch := make(chan providerOutcome, len(providersToQuery))
	want, maxPages := providerBudget(req.Limit, req.Offset)

	for _, pName := range providersToQuery {
		generation := breakerGenerations[pName]
		go func(name string, breakerGeneration uint64) {
			defer func() {
				if r := recover(); r != nil {
					if o.breakers != nil {
						o.breakers.Get(name).ReportFailureFor(breakerGeneration, fmt.Errorf("panic: %v", r))
					}
					ch <- providerOutcome{
						name: name,
						err:  fmt.Errorf("panic in provider %s: %v", name, r),
					}
				}
			}()

			p, ok := o.registry.Get(name)
			if !ok {
				if o.breakers != nil {
					o.breakers.Get(name).ReleaseFor(breakerGeneration)
				}
				ch <- providerOutcome{
					name: name,
					err:  fmt.Errorf("provider %q not found or not registered", name),
				}
				return
			}

			// Per-provider timeout bounded by parent context
			pCtx, cancel := context.WithTimeout(provCtx, o.timeout)
			defer cancel()

			pReq := model.ProviderRequest{
				Query:          req.Query,
				Limit:          want,
				MaxPages:       maxPages,
				Freshness:      req.Freshness,
				Country:        req.Country,
				Language:       req.Language,
				IncludeDomains: req.IncludeDomains,
				ExcludeDomains: req.ExcludeDomains,
				Timeout:        o.timeout,
			}

			providerStart := time.Now()
			resp, err := p.Search(pCtx, pReq)
			tookMs := time.Since(providerStart).Milliseconds()
			if err == nil && resp == nil {
				err = fmt.Errorf("provider %s returned an empty response", name)
			}
			if err != nil {
				if o.breakers != nil {
					if provCtx.Err() != nil || errors.Is(err, provider.ErrUnsupportedFilter) {
						// The caller went away, the overall deadline hit, this
						// provider was cut off as a straggler, or it cannot express
						// the requested filters; none of that says anything about
						// the provider's health.
						o.breakers.Get(name).ReleaseFor(breakerGeneration)
					} else {
						o.breakers.Get(name).ReportFailureFor(breakerGeneration, err)
					}
				}
				ch <- providerOutcome{name: name, err: err, tookMs: tookMs}
				return
			}
			if len(resp.Results) == 0 {
				o.logger.Warn("provider returned no parseable search results",
					slog.String("engine", name),
					slog.Duration("took", resp.Took),
				)
			}

			// The circuit breaker verdict for a successful fetch is reported by
			// the collector, after the response has passed the quality gate.
			ch <- providerOutcome{name: name, response: resp, tookMs: tookMs}
		}(pName, generation)
	}

	// 5. Collect outcomes with context cancellation resilience
	var successful []string
	failed := append([]model.ProviderError(nil), initialFailed...)
	var rawResponses []*model.ProviderResponse
	providerTimings := make(map[string]int64, len(providersToQuery))
	rawCount := 0
	replied := make(map[string]bool)
	received, withResults := 0, 0

	record := func(out providerOutcome) {
		received++
		replied[out.name] = true
		providerTimings[out.name] = out.tookMs
		if out.err != nil {
			o.logger.Warn("provider search failed",
				slog.String("engine", out.name),
				slog.Any("error", out.err),
			)
			failed = append(failed, model.ProviderError{Engine: out.name, Error: out.err.Error()})
			return
		}
		successful = append(successful, out.name)
		rawResponses = append(rawResponses, out.response)
		rawCount += len(out.response.Results)
		if len(out.response.Results) > 0 {
			withResults++
		}
	}
	markUnreplied := func(reason string) {
		for _, pName := range providersToQuery {
			if !replied[pName] {
				providerTimings[pName] = time.Since(start).Milliseconds()
				failed = append(failed, model.ProviderError{Engine: pName, Error: reason})
			}
		}
	}

	var graceTimer *time.Timer
	var grace <-chan time.Time
collectLoop:
	for received < len(providersToQuery) {
		select {
		case out := <-ch:
			record(out)
			outstanding := len(providersToQuery) - received
			if grace == nil && o.stragglerGrace > 0 && len(providersToQuery) >= 3 &&
				outstanding > 0 && withResults >= len(providersToQuery)-1 {
				graceTimer = time.NewTimer(o.stragglerGrace)
				grace = graceTimer.C
			}
		case <-grace:
			o.logger.Warn("cutting off slow provider after the others returned results",
				slog.Duration("grace", o.stragglerGrace),
			)
			cancelProviders()
			markUnreplied("skipped: still waiting after the other engines had returned results")
			break collectLoop
		case <-ctx.Done():
			// Request context cancelled or timed out
			o.logger.Warn("search request context terminated before all providers completed",
				slog.Any("error", ctx.Err()),
			)
			// Take any responses that are already available.
		drain:
			for {
				select {
				case out := <-ch:
					record(out)
				default:
					break drain
				}
			}
			markUnreplied(ctx.Err().Error())
			break collectLoop
		}
	}
	if graceTimer != nil {
		graceTimer.Stop()
	}
	if late := len(providersToQuery) - received; late > 0 {
		// Providers that are still running report their own failures. A late
		// success has nobody left to judge it, so settle its breaker here.
		go o.settleLateOutcomes(ch, late, breakerGenerations)
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
	uniqueResults, dedupedCount, dedupStats := pipeline.DeduplicateDetailed(rawResponses, dedupOpts)

	// Report circuit breaker verdicts now that the quality gate has run: a
	// provider that keeps serving off-topic pages is as unhealthy as one that
	// errors, and should be skipped for a while rather than queried every time.
	if o.breakers != nil {
		discarded := make(map[string]bool, len(dedupStats.LowQualityEngines))
		for _, engine := range dedupStats.LowQualityEngines {
			discarded[engine] = true
		}
		for _, name := range successful {
			breaker := o.breakers.Get(name)
			if discarded[name] {
				breaker.ReportFailureFor(breakerGenerations[name], errOffTopicResponse)
			} else {
				breaker.ReportSuccessFor(breakerGenerations[name])
			}
		}
	}

	// Providers whose whole SERP was off-topic (anti-bot result poisoning) are
	// reported as failed rather than successful.
	for _, engine := range dedupStats.LowQualityEngines {
		o.logger.Warn("discarding off-topic provider response", slog.String("engine", engine))
		successful = removeProvider(successful, engine)
		failed = append(failed, model.ProviderError{
			Engine: engine,
			Error:  "response discarded: results did not match the query (likely anti-bot result poisoning)",
		})
	}
	if len(dedupStats.LowQualityEngines) > 0 {
		sort.Slice(failed, func(i, j int) bool { return failed[i].Engine < failed[j].Engine })
	}
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
		extractCtx, cancelExtract := context.WithTimeout(ctx, o.extractTimeout)
		uniqueResults = o.extractor.ExtractBatch(extractCtx, uniqueResults, &extractOpts)
		cancelExtract()
	}

	// 9. Build the final unified response
	resp := model.NewSearchResponse(req.Query)
	resp.Results = uniqueResults
	resp.Metadata.TotalResults = len(uniqueResults)
	resp.Metadata.TookMs = time.Since(start).Milliseconds()
	resp.Metadata.Providers.Queried = queried
	resp.Metadata.Providers.Successful = successful
	resp.Metadata.Providers.Failed = failed
	resp.Metadata.Providers.TimingMs = providerTimings
	resp.Metadata.Stats.RawCount = rawCount
	resp.Metadata.Stats.DedupedCount = dedupedCount
	resp.Metadata.Stats.RelevanceRejectedCount = dedupStats.RelevanceRejected
	resp.Metadata.Stats.DomainRejectedCount = dedupStats.DomainRejected
	resp.Metadata.Stats.InvalidURLCount = dedupStats.InvalidURL
	resp.Metadata.Stats.DuplicateCount = dedupStats.Duplicates
	resp.Metadata.Stats.ProviderDiscardedCount = dedupStats.ProviderDiscarded

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

	// 10. Cache the response if we have results. A response missing one or
	// more providers is kept only briefly, so a transient failure does not pin
	// degraded results for the full TTL.
	if o.cache != nil && cacheKey != "" && len(resp.Results) > 0 {
		if len(failed) > 0 {
			o.cache.SetWithTTL(cacheKey, resp, degradedCacheTTL)
		} else {
			o.cache.Set(cacheKey, resp)
		}
	}

	return resp, nil
}

// degradedCacheTTL bounds how long a response with failed providers is cached.
const degradedCacheTTL = time.Minute

var errOffTopicResponse = errors.New("response did not match the query")

// resultsPerPage is roughly what one engine result page yields.
const resultsPerPage = 10

// maxProviderPages caps how many pages a provider fetches for one search.
const maxProviderPages = 3

// providerBudget returns how many results each provider should collect and
// how many pages it may fetch to cover the requested window (limit + offset).
// The default first page costs one request per engine; deeper windows fetch
// further pages because engines ignore requests for longer pages.
func providerBudget(limit, offset int) (want, maxPages int) {
	want = limit + offset
	if want < resultsPerPage {
		want = resultsPerPage
	}
	if want > model.MaxResultWindow {
		want = model.MaxResultWindow
	}
	maxPages = (want + resultsPerPage - 1) / resultsPerPage
	if maxPages > maxProviderPages {
		maxPages = maxProviderPages
	}
	return want, maxPages
}

// settleLateOutcomes consumes outcomes that arrive after the collector stopped
// waiting and closes the breaker for late successes, so a half-open probe
// slot is never left occupied.
func (o *Orchestrator) settleLateOutcomes(ch <-chan providerOutcome, n int, generations map[string]uint64) {
	for i := 0; i < n; i++ {
		out := <-ch
		if out.err == nil && o.breakers != nil {
			o.breakers.Get(out.name).ReportSuccessFor(generations[out.name])
		}
	}
}

func removeProvider(providers []string, name string) []string {
	out := providers[:0]
	for _, p := range providers {
		if p != name {
			out = append(out, p)
		}
	}
	return out
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
