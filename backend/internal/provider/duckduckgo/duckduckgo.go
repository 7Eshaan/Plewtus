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

// endpoint describes one of DuckDuckGo's two no-JavaScript result pages.
type endpoint struct {
	url, origin string
	parse       func(r io.Reader, limit int) ([]model.RawSearchResult, error)
}

var (
	htmlEndpoint = endpoint{HTMLURL, "https://html.duckduckgo.com", ParseHTMLResponse}
	liteEndpoint = endpoint{LiteURL, "https://lite.duckduckgo.com", ParseLiteResponse}
)

// Search executes a search query against DuckDuckGo: the /html/ endpoint
// first, falling back to /lite/ when it fails, then further pages from
// whichever endpoint answered while the request wants more results.
func (p *Provider) Search(ctx context.Context, req model.ProviderRequest) (*model.ProviderResponse, error) {
	start := time.Now()
	want, maxPages := provider.PageBudget(req)

	form := url.Values{}
	form.Set("q", provider.BuildSearchQuery(req))
	if df := mapFreshness(req.Freshness); df != "" {
		form.Set("df", df)
	}
	if kl := mapRegion(req.Country, req.Language); kl != "" {
		form.Set("kl", kl)
	}

	language := req.Language
	if language == "" {
		switch provider.NormalizeCountryCode(req.Country) {
		case "de", "fr", "es", "jp":
			language = provider.NormalizeCountryCode(req.Country)
		}
	}
	acceptLanguage := provider.AcceptLanguage(language)

	timeout := req.Timeout
	if timeout <= 0 {
		timeout = p.timeout
	}
	ddgCtx, cancelDDG := context.WithTimeout(ctx, timeout)
	defer cancelDDG()

	// The primary endpoint gets only half of the budget. Otherwise a hung
	// primary request would use the whole deadline and leave nothing for the
	// fallback.
	primaryCtx, cancelPrimary := context.WithTimeout(ddgCtx, timeout/2)
	active := htmlEndpoint
	results, body, err := p.queryEndpoint(primaryCtx, active, form, acceptLanguage)
	cancelPrimary()

	// Fall back when the primary failed, or returned nothing without showing
	// its genuine "no results" page (which suggests markup the parser missed).
	if err != nil || (len(results) == 0 && !isNoResultsPage(body)) {
		active = liteEndpoint
		liteResults, liteBody, liteErr := p.queryEndpoint(ddgCtx, active, form, acceptLanguage)
		switch {
		case liteErr == nil:
			results, body = liteResults, liteBody
		case err != nil:
			return nil, fmt.Errorf("duckduckgo search failed: %w (fallback error: %v)", err, liteErr)
		default:
			// The primary answered with an empty page and the fallback failed:
			// report the empty result rather than an error.
		}
	}

	var all []model.RawSearchResult
	seen := make(map[string]bool)
	all, _ = provider.AppendPage(all, seen, results)
	for page := 1; page < maxPages && len(all) < want; page++ {
		next, ok := ParseNextPageForm(body)
		if !ok {
			break
		}
		if next.Get("q") == "" {
			next.Set("q", form.Get("q"))
		}
		pageResults, pageBody, pageErr := p.queryEndpoint(ddgCtx, active, next, acceptLanguage)
		if pageErr != nil {
			break // keep what earlier pages returned
		}
		var added int
		if all, added = provider.AppendPage(all, seen, pageResults); added == 0 {
			break
		}
		body = pageBody
	}
	if len(all) > want {
		all = all[:want]
	}
	if all == nil {
		all = []model.RawSearchResult{}
	}

	return &model.ProviderResponse{Engine: p.Name(), Results: all, Took: time.Since(start)}, nil
}

// queryEndpoint posts the form and returns the parsed results together with
// the raw page (needed to follow DuckDuckGo's pagination form).
func (p *Provider) queryEndpoint(ctx context.Context, ep endpoint, form url.Values, acceptLanguage string) ([]model.RawSearchResult, []byte, error) {
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, ep.url, strings.NewReader(form.Encode()))
	if err != nil {
		return nil, nil, err
	}

	httpReq.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	httpReq.Header.Set("Origin", ep.origin)
	httpReq.Header.Set("Referer", ep.origin+"/")
	httpReq.Header.Set("Accept-Language", acceptLanguage)

	resp, err := p.client.Do(ctx, httpReq)
	if err != nil {
		return nil, nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusTooManyRequests {
		return nil, nil, fmt.Errorf("rate limited by duckduckgo (status 429)")
	}
	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusAccepted {
		return nil, nil, fmt.Errorf("unexpected status from duckduckgo: %d", resp.StatusCode)
	}

	bodyBytes, err := provider.ReadBody(resp)
	if err != nil {
		return nil, nil, fmt.Errorf("failed reading duckduckgo response body: %w", err)
	}

	results, err := ep.parse(bytes.NewReader(bodyBytes), 0)
	if err != nil {
		return nil, bodyBytes, err
	}
	return results, bodyBytes, nil
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
