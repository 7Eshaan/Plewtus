// Package brave queries Brave Search, which runs its own independent index.
//
// Two transports are supported:
//
//   - The official Search API (api.search.brave.com), used when an API key is
//     configured. This is the reliable route: Brave bills it by request (each
//     account gets a monthly credit) and it is not subject to bot checks.
//   - The public HTML results page (search.brave.com), used without a key. The
//     page is server-rendered, so results parse from plain HTML. Brave flags
//     most non-browser clients and answers with a JavaScript captcha (HTTP 429);
//     that is detected and reported, never worked around.
package brave

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"plewtus/internal/model"
	"plewtus/internal/provider"
)

const (
	// HTMLURL is the public results page.
	HTMLURL = "https://search.brave.com/search"
	// APIURL is the official web search endpoint.
	APIURL = "https://api.search.brave.com/res/v1/web/search"

	// maxOffset is the last page index both transports accept (pages 0-9).
	maxOffset = 9
	// apiPageSize is the largest page the API serves.
	apiPageSize = 20
)

// ErrChallengeEncountered reports Brave's captcha page instead of results.
var ErrChallengeEncountered = errors.New("brave served a captcha")

// Provider implements search through Brave.
type Provider struct {
	client  *provider.HTTPClient
	apiKey  string
	htmlURL string
	apiURL  string
}

// New creates a Brave provider. With an API key it uses the official API,
// otherwise the HTML results page.
func New(timeout time.Duration, userAgent, apiKey string) *Provider {
	return &Provider{
		client:  provider.NewHTTPClient(timeout, userAgent),
		apiKey:  strings.TrimSpace(apiKey),
		htmlURL: HTMLURL,
		apiURL:  APIURL,
	}
}

// Name returns the provider identifier.
func (p *Provider) Name() string {
	return "brave"
}

// Search fetches result pages until the request's budget is met.
func (p *Provider) Search(ctx context.Context, req model.ProviderRequest) (*model.ProviderResponse, error) {
	start := time.Now()
	want, maxPages := provider.PageBudget(req)

	freshness, err := mapFreshness(req.Freshness)
	if err != nil {
		return nil, err
	}

	var all []model.RawSearchResult
	seen := make(map[string]bool)
	for page := 0; page < maxPages && page <= maxOffset && len(all) < want; page++ {
		var results []model.RawSearchResult
		if p.apiKey != "" {
			results, err = p.fetchAPI(ctx, req, freshness, page)
		} else {
			results, err = p.fetchHTML(ctx, req, freshness, page)
		}
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
	return &model.ProviderResponse{Engine: p.Name(), Results: all, Took: time.Since(start)}, nil
}

// fetchHTML requests one page of the public results page.
func (p *Provider) fetchHTML(ctx context.Context, req model.ProviderRequest, freshness string, page int) ([]model.RawSearchResult, error) {
	params := url.Values{}
	params.Set("q", provider.BuildSearchQuery(req))
	params.Set("source", "web")
	if page > 0 {
		params.Set("offset", strconv.Itoa(page))
	}
	if freshness != "" {
		params.Set("tf", freshness)
	}

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodGet, p.htmlURL+"?"+params.Encode(), nil)
	if err != nil {
		return nil, err
	}
	lang := language(req.Language)
	httpReq.Header.Set("Accept-Language", provider.AcceptLanguage(lang))
	// Brave reads market and UI preferences from cookies on the HTML page.
	httpReq.Header.Set("Cookie", strings.Join([]string{
		"country=" + country(req.Country),
		"ui_lang=" + lang + "-" + country(req.Country),
		"safesearch=moderate",
		"useLocation=0",
		"summarizer=0",
	}, "; "))

	resp, err := p.client.Do(ctx, httpReq)
	if err != nil {
		return nil, fmt.Errorf("brave request failed: %w", err)
	}
	body, err := provider.ReadBody(resp)
	if err != nil {
		return nil, fmt.Errorf("failed reading brave response body: %w", err)
	}

	if isCaptcha(resp.StatusCode, body) {
		return nil, fmt.Errorf("%w (status %d; Brave blocks most non-browser clients, set PLEWTUS_BRAVE_API_KEY to use the official API)",
			ErrChallengeEncountered, resp.StatusCode)
	}
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected status from brave: %d", resp.StatusCode)
	}

	results, err := ParseHTMLResponse(bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("failed parsing brave response: %w", err)
	}
	return results, nil
}

// fetchAPI requests one page from the official Search API.
func (p *Provider) fetchAPI(ctx context.Context, req model.ProviderRequest, freshness string, page int) ([]model.RawSearchResult, error) {
	params := url.Values{}
	params.Set("q", provider.BuildSearchQuery(req))
	params.Set("count", strconv.Itoa(apiPageSize))
	if page > 0 {
		params.Set("offset", strconv.Itoa(page))
	}
	params.Set("country", strings.ToUpper(country(req.Country)))
	params.Set("search_lang", language(req.Language))
	params.Set("safesearch", "moderate")
	if freshness != "" {
		params.Set("freshness", freshness)
	}

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodGet, p.apiURL+"?"+params.Encode(), nil)
	if err != nil {
		return nil, err
	}
	httpReq.Header.Set("Accept", "application/json")
	httpReq.Header.Set("X-Subscription-Token", p.apiKey)

	resp, err := p.client.Do(ctx, httpReq)
	if err != nil {
		return nil, fmt.Errorf("brave api request failed: %w", err)
	}
	body, err := provider.ReadBody(resp)
	if err != nil {
		return nil, fmt.Errorf("failed reading brave api response body: %w", err)
	}

	switch {
	case resp.StatusCode == http.StatusTooManyRequests:
		return nil, fmt.Errorf("rate limited by brave api (status 429; per-second limit or monthly credit used up)")
	case resp.StatusCode == http.StatusUnauthorized || resp.StatusCode == http.StatusForbidden:
		return nil, fmt.Errorf("brave api rejected the API key (status %d)", resp.StatusCode)
	case resp.StatusCode == http.StatusUnprocessableEntity:
		return nil, fmt.Errorf("brave api rejected the request parameters (status 422)")
	case resp.StatusCode != http.StatusOK:
		return nil, fmt.Errorf("unexpected status from brave api: %d", resp.StatusCode)
	}

	results, err := ParseAPIResponse(body)
	if err != nil {
		return nil, fmt.Errorf("failed parsing brave api response: %w", err)
	}
	return results, nil
}

// isCaptcha recognises Brave's captcha page, which is served with HTTP 429
// and routes the SvelteKit app to /captcha.
func isCaptcha(status int, body []byte) bool {
	if status == http.StatusTooManyRequests {
		return true
	}
	lower := bytes.ToLower(body)
	return bytes.Contains(lower, []byte(`page:"/captcha"`)) ||
		bytes.Contains(lower, []byte("decided to schedule a captcha"))
}

// mapFreshness converts Plewtus freshness values to Brave's (the HTML page's
// tf and the API's freshness take the same codes).
func mapFreshness(freshness string) (string, error) {
	switch strings.ToLower(strings.TrimSpace(freshness)) {
	case "":
		return "", nil
	case "day":
		return "pd", nil
	case "week":
		return "pw", nil
	case "month":
		return "pm", nil
	case "year":
		return "py", nil
	default:
		return "", fmt.Errorf("%w: brave cannot filter by %q freshness", provider.ErrUnsupportedFilter, freshness)
	}
}

func language(lang string) string {
	lang = strings.ToLower(strings.TrimSpace(lang))
	if lang == "" {
		return "en"
	}
	return lang
}

func country(c string) string {
	c = provider.NormalizeCountryCode(c)
	if c == "" {
		return "us"
	}
	return c
}
