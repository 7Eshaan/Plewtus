package extractor

import (
	"context"
	"crypto/tls"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/cookiejar"
	"strings"
	"time"
)

// Fetcher safely retrieves web page HTML while enforcing SSRF protection,
// MIME type checks, and maximum size limits.
type Fetcher struct {
	client       *http.Client
	userAgent    string
	maxBodyBytes int64
}

// FetcherOptions configures the Fetcher.
type FetcherOptions struct {
	Timeout      time.Duration
	MaxBodyBytes int64
	UserAgent    string
	Transport    http.RoundTripper // Optional custom transport for testing
}

// Default limits
const (
	DefaultFetchTimeout = 5 * time.Second
	DefaultMaxBodyBytes = 2 * 1024 * 1024 // 2 MB
	DefaultUserAgent    = "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/128.0.0.0 Safari/537.36"
)

// NewFetcher creates a new safe Fetcher instance.
func NewFetcher(opts FetcherOptions) *Fetcher {
	timeout := opts.Timeout
	if timeout <= 0 {
		timeout = DefaultFetchTimeout
	}

	maxBytes := opts.MaxBodyBytes
	if maxBytes <= 0 {
		maxBytes = DefaultMaxBodyBytes
	}

	ua := opts.UserAgent
	if ua == "" {
		ua = DefaultUserAgent
	}

	jar, _ := cookiejar.New(nil)

	var transport http.RoundTripper = opts.Transport
	if transport == nil {
		transport = &http.Transport{
			DialContext:           SafeDialContext(timeout),
			TLSClientConfig:       &tls.Config{MinVersion: tls.VersionTLS12},
			ForceAttemptHTTP2:     true,
			MaxIdleConns:          100,
			IdleConnTimeout:       90 * time.Second,
			TLSHandshakeTimeout:   5 * time.Second,
			ExpectContinueTimeout: 1 * time.Second,
		}
	}

	client := &http.Client{
		Transport:     transport,
		CheckRedirect: SafeCheckRedirect(5),
		Jar:           jar,
		Timeout:       timeout,
	}

	return &Fetcher{
		client:       client,
		userAgent:    ua,
		maxBodyBytes: maxBytes,
	}
}

// Fetch safely downloads the HTML content from the given URL.
func (f *Fetcher) Fetch(ctx context.Context, targetURL string) ([]byte, string, error) {
	// 1. Initial URL validation against SSRF
	parsedURL, err := ValidateURL(targetURL)
	if err != nil {
		return nil, "", fmt.Errorf("url validation failed: %w", err)
	}

	// 2. Build request with polite headers
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, parsedURL.String(), nil)
	if err != nil {
		return nil, "", fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("User-Agent", f.userAgent)
	req.Header.Set("Accept", "text/html,application/xhtml+xml;q=0.9,*/*;q=0.8")
	req.Header.Set("Accept-Language", "en-US,en;q=0.9")
	req.Header.Set("Sec-Fetch-Dest", "document")
	req.Header.Set("Sec-Fetch-Mode", "navigate")
	req.Header.Set("Sec-Fetch-Site", "none")
	req.Header.Set("Sec-Fetch-User", "?1")

	// 3. Execute request
	resp, err := f.client.Do(req)
	if err != nil {
		return nil, "", fmt.Errorf("http request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, "", fmt.Errorf("upstream server responded with status %d %s", resp.StatusCode, resp.Status)
	}

	// 4. Validate Content-Type
	rawContentType := resp.Header.Get("Content-Type")
	contentType := strings.ToLower(rawContentType)

	// Read first chunk to detect MIME if header is missing or ambiguous
	limitReader := io.LimitReader(resp.Body, f.maxBodyBytes)
	initialBuf := make([]byte, 512)
	n, err := io.ReadFull(limitReader, initialBuf)
	if err != nil && !errors.Is(err, io.EOF) && !errors.Is(err, io.ErrUnexpectedEOF) {
		return nil, "", fmt.Errorf("failed to read response body: %w", err)
	}

	initialChunk := initialBuf[:n]

	if contentType == "" || strings.Contains(contentType, "application/octet-stream") {
		detected := http.DetectContentType(initialChunk)
		contentType = strings.ToLower(detected)
	}

	if !isHTMLContentType(contentType, initialChunk) {
		return nil, "", fmt.Errorf("unsupported content type %q: only HTML webpages are extracted", rawContentType)
	}

	// 5. Read remainder of the body up to maxBodyBytes
	var remainder []byte
	if n > 0 && n == 512 {
		rest, err := io.ReadAll(limitReader)
		if err != nil {
			return nil, "", fmt.Errorf("failed reading response body: %w", err)
		}
		remainder = rest
	}

	body := append(initialChunk, remainder...)
	return body, contentType, nil
}

func isHTMLContentType(contentType string, chunk []byte) bool {
	if strings.Contains(contentType, "text/html") || strings.Contains(contentType, "application/xhtml+xml") {
		return true
	}

	// If header is plain text or generic XML, check if content looks like HTML
	lower := strings.ToLower(string(chunk))
	if strings.Contains(lower, "<!doctype html") ||
		strings.Contains(lower, "<html") ||
		strings.Contains(lower, "<head") ||
		strings.Contains(lower, "<body") {
		return true
	}

	return false
}
