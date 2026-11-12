// Package marginalia queries Marginalia Search, an independent index of the
// small, non-commercial web.
//
// Marginalia's HTML result pages sit behind a bot-throttling interstitial
// ("Wait A Moment", with an sst token and a countdown). Rather than working
// around that gate, this provider uses Marginalia's official JSON API
// (api2.marginalia-search.com), which the project offers for programmatic use.
// The shared "public" key works without signup but has a low, shared
// queries-per-minute limit; a personal key can be configured instead.
// Results are licensed CC-BY-NC-SA 4.0 under free keys.
package marginalia

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"plewtus/internal/model"
	"plewtus/internal/provider"
)

// BaseURL is the official v2 search endpoint.
const BaseURL = "https://api2.marginalia-search.com/search"

// PublicAPIKey is the shared key Marginalia publishes for experimentation.
const PublicAPIKey = "public"

// maxCount is the largest page the API serves in one request.
const maxCount = 100

// Provider implements search through the Marginalia Search API.
type Provider struct {
	client  *provider.HTTPClient
	apiKey  string
	baseURL string
	now     func() time.Time
}

// New creates a Marginalia provider. An empty apiKey uses the shared public key.
func New(timeout time.Duration, userAgent, apiKey string) *Provider {
	apiKey = strings.TrimSpace(apiKey)
	if apiKey == "" {
		apiKey = PublicAPIKey
	}
	return &Provider{
		client:  provider.NewHTTPClient(timeout, userAgent),
		apiKey:  apiKey,
		baseURL: BaseURL,
		now:     time.Now,
	}
}

// Name returns the provider identifier.
func (p *Provider) Name() string {
	return "marginalia"
}

// Search runs one API request. The API serves up to 100 results per page, so
// the whole window the orchestrator asks for (at most 50) fits in one call.
func (p *Provider) Search(ctx context.Context, req model.ProviderRequest) (*model.ProviderResponse, error) {
	start := time.Now()

	query, err := p.buildQuery(req)
	if err != nil {
		return nil, err
	}
	want, _ := provider.PageBudget(req)
	if want > maxCount {
		want = maxCount
	}

	params := url.Values{}
	params.Set("query", query)
	params.Set("count", strconv.Itoa(want))
	// "lang" is read by the v2 handler (default "en") though not in the docs.
	if lang := strings.ToLower(strings.TrimSpace(req.Language)); lang != "" {
		params.Set("lang", lang)
	}

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodGet, p.baseURL+"?"+params.Encode(), nil)
	if err != nil {
		return nil, err
	}
	httpReq.Header.Set("API-Key", p.apiKey)
	httpReq.Header.Set("Accept", "application/json")

	resp, err := p.client.Do(ctx, httpReq)
	if err != nil {
		return nil, fmt.Errorf("marginalia request failed: %w", err)
	}
	defer resp.Body.Close()

	switch {
	case resp.StatusCode == http.StatusTooManyRequests || resp.StatusCode == http.StatusServiceUnavailable:
		hint := ""
		if p.apiKey == PublicAPIKey {
			hint = "; the shared public key is often exhausted, set PLEWTUS_MARGINALIA_API_KEY"
		}
		return nil, fmt.Errorf("rate limited by marginalia (status %d%s)", resp.StatusCode, hint)
	case resp.StatusCode == http.StatusUnauthorized || resp.StatusCode == http.StatusForbidden:
		return nil, fmt.Errorf("marginalia rejected the API key (status %d)", resp.StatusCode)
	case resp.StatusCode == http.StatusGatewayTimeout:
		return nil, fmt.Errorf("marginalia query timed out upstream (status 504)")
	case resp.StatusCode != http.StatusOK:
		return nil, fmt.Errorf("unexpected status from marginalia: %d", resp.StatusCode)
	}

	body, err := provider.ReadBody(resp)
	if err != nil {
		return nil, fmt.Errorf("failed reading marginalia response body: %w", err)
	}
	results, err := ParseResponse(body)
	if err != nil {
		return nil, fmt.Errorf("failed parsing marginalia response: %w", err)
	}
	if len(results) > want {
		results = results[:want]
	}

	return &model.ProviderResponse{Engine: p.Name(), Results: results, Took: time.Since(start)}, nil
}

// buildQuery translates the request into Marginalia query syntax. Marginalia
// has no OR operator (parentheses mean "unweighted terms") and only
// year-granular dates, so filters it cannot express exactly are either left to
// the pipeline's post-filtering (domains) or rejected as unsupported.
func (p *Provider) buildQuery(req model.ProviderRequest) (string, error) {
	parts := []string{strings.TrimSpace(req.Query)}

	// A single domain narrows upstream; several are enforced by the pipeline's
	// strict domain post-filter instead. "site:*.d" covers d and its subdomains.
	if len(req.IncludeDomains) == 1 {
		parts = append(parts, "site:*."+req.IncludeDomains[0])
	}

	switch strings.ToLower(strings.TrimSpace(req.Freshness)) {
	case "":
	case "year":
		// year>N means "published in or after N"; the closest match for the
		// past twelve months is "this calendar year or last".
		parts = append(parts, "year>"+strconv.Itoa(p.now().UTC().Year()-1))
	default:
		return "", fmt.Errorf("%w: marginalia cannot filter by %q freshness (only 'year')", provider.ErrUnsupportedFilter, req.Freshness)
	}

	return strings.Join(parts, " "), nil
}
