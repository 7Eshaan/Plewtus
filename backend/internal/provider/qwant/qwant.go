// Package qwant queries Qwant, a European search engine.
//
// Qwant's pages are a JavaScript app fed by a JSON API (api.qwant.com/v3),
// and its old no-JavaScript "lite" site now redirects to the main app, so
// this provider reads that API with one ordinary request per page.
//
// Two limits apply that this provider reports rather than works around:
//
//   - Qwant only serves some countries. Elsewhere the API answers
//     error_code 50 (the site says "not yet available in your country").
//   - The API sits behind DataDome bot protection, which may answer with a
//     captcha (a top-level "url" in the JSON, often with HTTP 403).
package qwant

import (
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

// BaseURL is the web search endpoint of Qwant's API.
const BaseURL = "https://api.qwant.com/v3/search/web"

const (
	pageSize = 10
	// maxPages is the deepest Qwant serves (offset 40).
	maxPages = 5
)

// API error codes seen in Qwant's {"status":"error","data":{"error_code":N}} replies.
const (
	errCodeRateLimited  = 24
	errCodeNotInCountry = 50
)

var (
	// ErrChallengeEncountered reports a DataDome captcha instead of results.
	ErrChallengeEncountered = errors.New("qwant served a captcha (DataDome bot protection)")
	// ErrNotAvailable reports Qwant's country restriction.
	ErrNotAvailable = errors.New("qwant is not available in this server's country (error_code 50)")
)

// Provider implements search through Qwant's API.
type Provider struct {
	client  *provider.HTTPClient
	baseURL string
}

// New creates a Qwant provider.
func New(timeout time.Duration, userAgent string) *Provider {
	return &Provider{
		client:  provider.NewHTTPClient(timeout, userAgent),
		baseURL: BaseURL,
	}
}

// Name returns the provider identifier.
func (p *Provider) Name() string {
	return "qwant"
}

// Search fetches result pages until the request's budget is met.
func (p *Provider) Search(ctx context.Context, req model.ProviderRequest) (*model.ProviderResponse, error) {
	start := time.Now()
	if strings.TrimSpace(req.Freshness) != "" {
		return nil, fmt.Errorf("%w: qwant's API has no date filter", provider.ErrUnsupportedFilter)
	}
	want, pages := provider.PageBudget(req)

	var all []model.RawSearchResult
	seen := make(map[string]bool)
	for page := 0; page < pages && page < maxPages && len(all) < want; page++ {
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
	return &model.ProviderResponse{Engine: p.Name(), Results: all, Took: time.Since(start)}, nil
}

func (p *Provider) fetchPage(ctx context.Context, req model.ProviderRequest, page int) ([]model.RawSearchResult, error) {
	params := url.Values{}
	params.Set("q", buildQuery(req))
	params.Set("count", strconv.Itoa(pageSize))
	params.Set("offset", strconv.Itoa(page*pageSize))
	params.Set("locale", Locale(req.Language, req.Country))
	params.Set("device", "desktop")
	params.Set("safesearch", "1")

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodGet, p.baseURL+"?"+params.Encode(), nil)
	if err != nil {
		return nil, err
	}
	httpReq.Header.Set("Accept", "application/json, text/plain, */*")
	httpReq.Header.Set("Origin", "https://www.qwant.com")
	httpReq.Header.Set("Referer", "https://www.qwant.com/")

	resp, err := p.client.Do(ctx, httpReq)
	if err != nil {
		return nil, fmt.Errorf("qwant request failed: %w", err)
	}
	body, err := provider.ReadBody(resp)
	if err != nil {
		return nil, fmt.Errorf("failed reading qwant response body: %w", err)
	}

	// Qwant explains most failures in the JSON body, whatever the HTTP status.
	parsed, parseErr := ParseResponse(body)
	if parseErr == nil {
		if parsed.Captcha {
			return nil, fmt.Errorf("%w (status %d)", ErrChallengeEncountered, resp.StatusCode)
		}
		switch parsed.ErrorCode {
		case 0:
		case errCodeNotInCountry:
			return nil, ErrNotAvailable
		case errCodeRateLimited:
			return nil, fmt.Errorf("rate limited by qwant (error_code 24)")
		default:
			return nil, fmt.Errorf("qwant api error (error_code %d, status %d)", parsed.ErrorCode, resp.StatusCode)
		}
	}

	switch {
	case resp.StatusCode == http.StatusTooManyRequests:
		return nil, fmt.Errorf("rate limited by qwant (status 429)")
	case resp.StatusCode == http.StatusForbidden:
		return nil, fmt.Errorf("%w (status 403)", ErrChallengeEncountered)
	case resp.StatusCode != http.StatusOK:
		return nil, fmt.Errorf("unexpected status from qwant: %d", resp.StatusCode)
	case parseErr != nil:
		return nil, fmt.Errorf("failed parsing qwant response: %w", parseErr)
	}
	return parsed.Results, nil
}

// buildQuery adds a site: filter for a single include domain. Several
// include domains and all excludes are left to the pipeline's post-filter,
// since Qwant's operator support is not documented.
func buildQuery(req model.ProviderRequest) string {
	q := strings.TrimSpace(req.Query)
	if len(req.IncludeDomains) == 1 {
		q += " site:" + req.IncludeDomains[0]
	}
	return q
}

// defaultCountry pairs a language with the country Qwant expects for it.
var defaultCountry = map[string]string{
	"en": "US", "fr": "FR", "de": "DE", "es": "ES", "it": "IT", "nl": "NL", "pt": "PT", "pl": "PL",
}

// Locale builds Qwant's locale parameter (e.g. "en_US", "fr_FR") from the
// request's language and country, defaulting to en_US.
func Locale(language, country string) string {
	lang := strings.ToLower(strings.TrimSpace(language))
	if lang == "" {
		lang = "en"
	}
	cc := strings.ToUpper(provider.NormalizeCountryCode(country))
	if cc == "" {
		cc = defaultCountry[lang]
	}
	if cc == "" {
		cc = "US"
	}
	return lang + "_" + cc
}
