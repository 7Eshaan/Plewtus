package bing

import (
	"bytes"
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

const BaseURL = "https://www.bing.com/search"

// Provider implements search scraping for Bing.
type Provider struct {
	client *provider.HTTPClient
}

// New creates a new Bing Provider instance.
func New(timeout time.Duration, userAgent string) *Provider {
	return &Provider{
		client: provider.NewHTTPClient(timeout, userAgent),
	}
}

// Name returns the provider identifier.
func (p *Provider) Name() string {
	return "bing"
}

// Search executes a search query against Bing, fetching further result
// pages while the request wants more results than have been collected.
func (p *Provider) Search(ctx context.Context, req model.ProviderRequest) (*model.ProviderResponse, error) {
	start := time.Now()
	want, maxPages := provider.PageBudget(req)

	var all []model.RawSearchResult
	seen := make(map[string]bool)
	for page := 0; page < maxPages && len(all) < want; page++ {
		results, err := p.fetchPage(ctx, req, page)
		if err != nil {
			if page == 0 {
				return nil, err
			}
			break // keep what earlier pages returned
		}
		var added int
		if all, added = provider.AppendPage(all, seen, results); added == 0 {
			break
		}
	}
	if len(all) > want {
		all = all[:want]
	}

	// Zero organic results is a legitimate outcome (e.g. narrow site: filters)
	// and must not count as a provider failure. Block pages are reported by the
	// parser as ErrChallengeEncountered instead.
	return &model.ProviderResponse{Engine: p.Name(), Results: all, Took: time.Since(start)}, nil
}

// fetchPage requests a zero-based result page. Bing numbers pages in steps of
// ten regardless of how many organic results a page actually showed.
func (p *Provider) fetchPage(ctx context.Context, req model.ProviderRequest, page int) ([]model.RawSearchResult, error) {
	params := url.Values{}
	params.Set("q", provider.BuildSearchQuery(req))
	if page > 0 {
		params.Set("first", strconv.Itoa(page*10+1))
		params.Set("FORM", "PERE")
	}

	lang := req.Language
	if lang == "" {
		lang = "en"
	}
	params.Set("setlang", strings.ToLower(lang))

	country := req.Country
	if country == "" {
		// Default to US region when no country is specified to prevent
		// unauthenticated IP-localized noise from hijacking general search queries.
		country = "US"
	}
	params.Set("cc", strings.ToUpper(provider.NormalizeCountryCode(country)))
	if filter := mapFreshness(req.Freshness); filter != "" {
		params.Set("filters", filter)
	}

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodGet, BaseURL+"?"+params.Encode(), nil)
	if err != nil {
		return nil, err
	}
	httpReq.Header.Set("Accept-Language", provider.AcceptLanguage(lang))
	httpReq.Header.Set("Referer", "https://www.bing.com/")

	resp, err := p.client.Do(ctx, httpReq)
	if err != nil {
		return nil, fmt.Errorf("bing request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusTooManyRequests {
		return nil, fmt.Errorf("rate limited by bing (status 429)")
	}
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected status from bing: %d", resp.StatusCode)
	}

	bodyBytes, err := provider.ReadBody(resp)
	if err != nil {
		return nil, fmt.Errorf("failed reading bing response body: %w", err)
	}
	results, err := ParseHTMLResponse(bytes.NewReader(bodyBytes), 0)
	if err != nil {
		return nil, fmt.Errorf("failed parsing bing response: %w", err)
	}
	return results, nil
}

func mapFreshness(freshness string) string {
	switch strings.ToLower(strings.TrimSpace(freshness)) {
	case "day":
		return `ex1:"ez1"`
	case "week":
		return `ex1:"ez2"`
	case "month":
		return `ex1:"ez3"`
	case "year":
		return `ex1:"ez5"`
	default:
		return ""
	}
}
