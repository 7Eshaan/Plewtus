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
		client: provider.NewHTTPClient(timeout, userAgent),
	}
}

// Name returns the provider identifier.
func (p *Provider) Name() string {
	return "yahoo"
}

// Search executes a search query against Yahoo.
func (p *Provider) Search(ctx context.Context, req model.ProviderRequest) (*model.ProviderResponse, error) {
	start := time.Now()

	query := buildQuery(req)
	params := url.Values{}
	params.Set("p", query)

	if req.Limit > 0 {
		params.Set("n", strconv.Itoa(req.Limit))
	}
	if req.Language != "" {
		params.Set("vl", "lang_"+strings.ToLower(req.Language))
	}
	if req.Country != "" {
		params.Set("vc", strings.ToLower(req.Country))
	}
	if age := mapFreshness(req.Freshness); age != "" {
		params.Set("age", age)
	}

	searchURL := BaseURL + "?" + params.Encode()

	var resp *http.Response
	var err error

	// Retry once on transient network errors (such as unexpected EOF during Yahoo's initial cookie redirect)
	for attempt := 0; attempt < 2; attempt++ {
		if attempt > 0 {
			time.Sleep(300 * time.Millisecond)
		}

		var httpReq *http.Request
		httpReq, err = http.NewRequestWithContext(ctx, http.MethodGet, searchURL, nil)
		if err != nil {
			return nil, err
		}

		httpReq.Header.Set("Accept", "text/html,application/xhtml+xml,application/xml;q=0.9,image/avif,image/webp,*/*;q=0.8")
		httpReq.Header.Set("Accept-Language", "en-US,en;q=0.9")
		httpReq.Header.Set("Referer", "https://search.yahoo.com/")

		resp, err = p.client.Do(ctx, httpReq)
		if err == nil {
			break
		}
		if !isRetryable(err) {
			break
		}
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

	results, err := ParseHTMLResponse(bytes.NewReader(bodyBytes), req.Limit)
	if err != nil {
		return nil, fmt.Errorf("failed parsing yahoo response: %w", err)
	}

	return &model.ProviderResponse{
		Engine:  p.Name(),
		Results: results,
		Took:    time.Since(start),
	}, nil
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
		return "1d"
	case "week":
		return "1w"
	case "month":
		return "1m"
	default:
		return ""
	}
}
