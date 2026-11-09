package duckduckgo

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"plewtus/internal/model"
	"plewtus/internal/provider"
)

const (
	HTMLURL = "https://html.duckduckgo.com/html/"
	LiteURL = "https://lite.duckduckgo.com/lite/"
)

// Provider implements search scraping for DuckDuckGo.
type Provider struct {
	client  *provider.HTTPClient
	timeout time.Duration
}

// New creates a new DuckDuckGo Provider instance.
func New(timeout time.Duration, userAgent string) *Provider {
	return &Provider{
		client:  provider.NewHTTPClient(timeout, userAgent),
		timeout: timeout,
	}
}

// Name returns the provider identifier.
func (p *Provider) Name() string {
	return "duckduckgo"
}

// Search executes a search query against DuckDuckGo.
func (p *Provider) Search(ctx context.Context, req model.ProviderRequest) (*model.ProviderResponse, error) {
	start := time.Now()

	// 1. Build query string with domain filters if specified
	query := provider.BuildSearchQuery(req)

	// 2. Build form parameters
	form := url.Values{}
	form.Set("q", query)

	if df := mapFreshness(req.Freshness); df != "" {
		form.Set("df", df)
	}
	if kl := mapRegion(req.Country, req.Language); kl != "" {
		form.Set("kl", kl)
	}

	// 3. Try primary endpoint (html.duckduckgo.com/html/) with only part of
	// the provider budget. Otherwise a hung primary request consumes the entire
	// context deadline and the lite fallback is invoked with an already-expired
	// context, making fallback impossible.
	timeout := req.Timeout
	if timeout <= 0 {
		timeout = p.timeout
	}
	// Cap DuckDuckGo's total budget to 3s so a hung/rate-limited datacenter request
	// does not block the entire search for the full server timeout.
	const maxDDGBudget = 3 * time.Second
	if timeout > maxDDGBudget {
		timeout = maxDDGBudget
	}
	ddgCtx, cancelDDG := context.WithTimeout(ctx, timeout)
	defer cancelDDG()

	primaryBudget := timeout / 2
	if primaryBudget <= 0 {
		primaryBudget = 1200 * time.Millisecond
	}
	primaryCtx, cancelPrimary := context.WithTimeout(ddgCtx, primaryBudget)
	language := req.Language
	if language == "" {
		switch provider.NormalizeCountryCode(req.Country) {
		case "de", "fr", "es", "jp":
			language = provider.NormalizeCountryCode(req.Country)
		}
	}
	acceptLanguage := provider.AcceptLanguage(language)
	results, err := p.queryEndpoint(primaryCtx, HTMLURL, "https://html.duckduckgo.com", form, acceptLanguage, req.Limit, ParseHTMLResponse)
	cancelPrimary()
	if err == nil && len(results) > 0 {
		return &model.ProviderResponse{
			Engine:  p.Name(),
			Results: results,
			Took:    time.Since(start),
		}, nil
	}

	// 4. If primary failed or returned challenge, try fallback endpoint (lite.duckduckgo.com/lite/)
	liteResults, liteErr := p.queryEndpoint(ddgCtx, LiteURL, "https://lite.duckduckgo.com", form, acceptLanguage, req.Limit, ParseLiteResponse)
	if liteErr == nil && len(liteResults) > 0 {
		return &model.ProviderResponse{
			Engine:  p.Name(),
			Results: liteResults,
			Took:    time.Since(start),
		}, nil
	}

	// If both failed, report the failure
	if err != nil {
		return nil, fmt.Errorf("duckduckgo search failed: %w (fallback error: %v)", err, liteErr)
	}
	if liteErr != nil {
		return nil, fmt.Errorf("duckduckgo fallback search failed: %w", liteErr)
	}

	// Both endpoints answered without a challenge but had no organic results.
	return &model.ProviderResponse{Engine: p.Name(), Results: []model.RawSearchResult{}, Took: time.Since(start)}, nil
}

func (p *Provider) queryEndpoint(
	ctx context.Context,
	endpointURL, origin string,
	form url.Values,
	acceptLanguage string,
	limit int,
	parseFunc func(r io.Reader, limit int) ([]model.RawSearchResult, error),
) ([]model.RawSearchResult, error) {
	formData := form.Encode()
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, endpointURL, strings.NewReader(formData))
	if err != nil {
		return nil, err
	}

	httpReq.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	httpReq.Header.Set("Origin", origin)
	httpReq.Header.Set("Referer", origin+"/")
	httpReq.Header.Set("Accept-Language", acceptLanguage)

	resp, err := p.client.Do(ctx, httpReq)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusTooManyRequests {
		return nil, fmt.Errorf("rate limited by duckduckgo (status 429)")
	}
	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusAccepted {
		return nil, fmt.Errorf("unexpected status from duckduckgo: %d", resp.StatusCode)
	}

	bodyBytes, err := provider.ReadBody(resp)
	if err != nil {
		return nil, fmt.Errorf("failed reading duckduckgo response body: %w", err)
	}

	return parseFunc(bytes.NewReader(bodyBytes), limit)
}

func mapFreshness(freshness string) string {
	switch strings.ToLower(strings.TrimSpace(freshness)) {
	case "day":
		return "d"
	case "week":
		return "w"
	case "month":
		return "m"
	case "year":
		return "y"
	default:
		return ""
	}
}

func mapRegion(country, language string) string {
	c := provider.NormalizeCountryCode(country)
	l := strings.ToLower(strings.TrimSpace(language))
	if c == "" {
		c = "us"
	}
	if l == "" {
		switch c {
		case "de", "fr", "es", "jp":
			l = c
		default:
			l = "en"
		}
	}
	if c == "gb" {
		c = "uk" // DuckDuckGo's locale identifier uses "uk" for Great Britain.
	}
	return c + "-" + l
}
