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
	client *provider.HTTPClient
}

// New creates a new DuckDuckGo Provider instance.
func New(timeout time.Duration, userAgent string) *Provider {
	return &Provider{
		client: provider.NewHTTPClient(timeout, userAgent),
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
	query := buildQuery(req)

	// 2. Build form parameters
	form := url.Values{}
	form.Set("q", query)

	if df := mapFreshness(req.Freshness); df != "" {
		form.Set("df", df)
	}
	if kl := mapRegion(req.Country, req.Language); kl != "" {
		form.Set("kl", kl)
	}

	// 3. Try primary endpoint (html.duckduckgo.com/html/)
	results, err := p.queryEndpoint(ctx, HTMLURL, "https://html.duckduckgo.com", form, req.Limit, ParseHTMLResponse)
	if err == nil && len(results) > 0 {
		return &model.ProviderResponse{
			Engine:  p.Name(),
			Results: results,
			Took:    time.Since(start),
		}, nil
	}

	// 4. If primary failed or returned challenge, try fallback endpoint (lite.duckduckgo.com/lite/)
	liteResults, liteErr := p.queryEndpoint(ctx, LiteURL, "https://lite.duckduckgo.com", form, req.Limit, ParseLiteResponse)
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

	return &model.ProviderResponse{
		Engine:  p.Name(),
		Results: []model.RawSearchResult{},
		Took:    time.Since(start),
	}, nil
}

func (p *Provider) queryEndpoint(
	ctx context.Context,
	endpointURL, origin string,
	form url.Values,
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


func buildQuery(req model.ProviderRequest) string {
	var parts []string
	if trimmed := strings.TrimSpace(req.Query); trimmed != "" {
		parts = append(parts, trimmed)
	}
	for _, inc := range req.IncludeDomains {
		parts = append(parts, "site:"+inc)
	}
	for _, exc := range req.ExcludeDomains {
		parts = append(parts, "-site:"+exc)
	}
	return strings.Join(parts, " ")
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
	c := strings.ToLower(strings.TrimSpace(country))
	l := strings.ToLower(strings.TrimSpace(language))

	if c != "" && l != "" {
		return c + "-" + l
	}
	if c != "" {
		switch c {
		case "us":
			return "us-en"
		case "uk", "gb":
			return "uk-en"
		case "de":
			return "de-de"
		case "fr":
			return "fr-fr"
		case "es":
			return "es-es"
		case "ca":
			return "ca-en"
		case "au":
			return "au-en"
		case "in":
			return "in-en"
		case "jp":
			return "jp-jp"
		default:
			return c + "-en"
		}
	}
	return ""
}
