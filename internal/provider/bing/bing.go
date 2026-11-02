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

// Search executes a search query against Bing.
func (p *Provider) Search(ctx context.Context, req model.ProviderRequest) (*model.ProviderResponse, error) {
	start := time.Now()

	query := buildQuery(req)
	params := url.Values{}
	params.Set("q", query)

	if req.Limit > 0 {
		params.Set("count", strconv.Itoa(req.Limit))
	}
	if req.Language != "" {
		params.Set("setlang", strings.ToLower(req.Language))
	}
	if req.Country != "" {
		params.Set("cc", strings.ToUpper(req.Country))
	}
	if filter := mapFreshness(req.Freshness); filter != "" {
		params.Set("filters", filter)
	}

	searchURL := BaseURL + "?" + params.Encode()
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodGet, searchURL, nil)
	if err != nil {
		return nil, err
	}

	httpReq.Header.Set("Accept", "text/html,application/xhtml+xml,application/xml;q=0.9,image/avif,image/webp,*/*;q=0.8")
	httpReq.Header.Set("Accept-Language", "en-US,en;q=0.9")
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

	results, err := ParseHTMLResponse(bytes.NewReader(bodyBytes), req.Limit)
	if err != nil {
		return nil, fmt.Errorf("failed parsing bing response: %w", err)
	}

	return &model.ProviderResponse{
		Engine:  p.Name(),
		Results: results,
		Took:    time.Since(start),
	}, nil
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
		return `ex1:"ez1"`
	case "week":
		return `ex1:"ez2"`
	case "month":
		return `ex1:"ez3"`
	default:
		return ""
	}
}
