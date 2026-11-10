package yahoo

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"plewtus/internal/model"
	"plewtus/internal/provider"
)

const BaseURL = "https://search.yahoo.com/search"

// Provider implements search scraping for Yahoo.
type Provider struct {
	client *provider.HTTPClient
}

// New creates a new Yahoo Provider instance.
func New(timeout time.Duration, userAgent string) *Provider {
	return &Provider{
		// Yahoo's edge drops HTTP/2 connections from Go's TLS stack (unexpected EOF).
		client: provider.NewHTTPClient(timeout, userAgent, provider.WithHTTP1Only()),
	}
}

// Name returns the provider identifier.
func (p *Provider) Name() string {
	return "yahoo"
}

// Search executes a search query against Yahoo, fetching further result
// pages while the request wants more results than have been collected.
func (p *Provider) Search(ctx context.Context, req model.ProviderRequest) (*model.ProviderResponse, error) {
	start := time.Now()
	want, maxPages := provider.PageBudget(req)

	var all []model.RawSearchResult
	seen := make(map[string]bool)
	for page := 0; page < maxPages && len(all) < want; page++ {
		results, err := p.fetchPage(ctx, req, len(all))
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

// fetchPage requests the result page that starts after `collected` results.
func (p *Provider) fetchPage(ctx context.Context, req model.ProviderRequest, collected int) ([]model.RawSearchResult, error) {
	params := url.Values{}
	params.Set("p", provider.BuildSearchQuery(req))
	if collected > 0 {
		params.Set("b", strconv.Itoa(collected+1)) // 1-based index of the first result
	}

	lang := req.Language
	if lang == "" {
		lang = "en"
	}
	params.Set("vl", "lang_"+strings.ToLower(lang))

	country := req.Country
	if country == "" {
		country = "us"
	}
	params.Set("vc", provider.NormalizeCountryCode(country))

	if age := mapFreshness(req.Freshness); age != "" {
		params.Set("age", age)
	}

	searchURL := BaseURL + "?" + params.Encode()

	var resp *http.Response
	var err error

	// Retry on transient network errors (such as unexpected EOF during Yahoo's initial cookie redirect)
	for attempt := 0; attempt < 3; attempt++ {
		if attempt > 0 {
			timer := time.NewTimer(time.Duration(attempt) * 400 * time.Millisecond)
			select {
			case <-ctx.Done():
				timer.Stop()
				return nil, ctx.Err()
			case <-timer.C:
			}
		}

		var httpReq *http.Request
		httpReq, err = http.NewRequestWithContext(ctx, http.MethodGet, searchURL, nil)
		if err != nil {
			return nil, err
		}
		httpReq.Header.Set("Accept-Language", provider.AcceptLanguage(lang))
		httpReq.Header.Set("Referer", "https://search.yahoo.com/")

		resp, err = p.client.Do(ctx, httpReq)
		if err == nil {
			if resp.StatusCode >= 500 && attempt < 2 {
				_ = resp.Body.Close()
				p.client.CloseIdleConnections()
				continue
			}
			break
		}
		if !isRetryable(err) {
			break
		}
		p.client.CloseIdleConnections()
	}

	if err != nil {
		return nil, fmt.Errorf("yahoo request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusTooManyRequests {
		return nil, fmt.Errorf("rate limited by yahoo (status 429)")
	}
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected status from yahoo: %d", resp.StatusCode)
	}

	bodyBytes, err := provider.ReadBody(resp)
	if err != nil {
		return nil, fmt.Errorf("failed reading yahoo response body: %w", err)
	}
	results, err := ParseHTMLResponse(bytes.NewReader(bodyBytes), 0)
	if err != nil {
		return nil, fmt.Errorf("failed parsing yahoo response: %w", err)
	}
	return results, nil
}

func isRetryable(err error) bool {
	if err == nil {
		return false
	}
	if errors.Is(err, io.EOF) || errors.Is(err, io.ErrUnexpectedEOF) {
		return true
	}
	msg := strings.ToLower(err.Error())
	return strings.Contains(msg, "eof") ||
		strings.Contains(msg, "connection reset") ||
		strings.Contains(msg, "broken pipe")
}

func mapFreshness(freshness string) string {
	switch strings.ToLower(strings.TrimSpace(freshness)) {
	case "day":
		return "1d"
	case "week":
		return "1w"
	case "month":
		return "1m"
	case "year":
		return "1y"
	default:
		return ""
	}
}
