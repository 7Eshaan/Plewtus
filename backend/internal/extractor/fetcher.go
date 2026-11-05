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

	// 4. Read only a small prefix before MIME validation. Unsupported files
	// should not make the extractor download their entire body just to reject it.
	prefixLimit := int64(512)
	if f.maxBodyBytes <= prefixLimit {
		prefixLimit = f.maxBodyBytes + 1
	}
	prefix := make([]byte, prefixLimit)
	n, readErr := io.ReadFull(resp.Body, prefix)
	if readErr != nil && !errors.Is(readErr, io.EOF) && !errors.Is(readErr, io.ErrUnexpectedEOF) {
		return nil, "", fmt.Errorf("failed reading response body: %w", readErr)
	}
	if int64(n) > f.maxBodyBytes {
		return nil, "", fmt.Errorf("response body exceeds %d bytes", f.maxBodyBytes)
	}
	if n == 0 {
		return nil, "", errors.New("response body is empty")
	}
	initialChunk := prefix[:n]

	// 5. Validate Content-Type. Trust explicit web MIME types, sniff generic or
	// missing types, and reject other declared types before reading more bytes.
	rawContentType := resp.Header.Get("Content-Type")
	contentType := strings.ToLower(strings.TrimSpace(rawContentType))
	baseContentType := strings.ToLower(strings.TrimSpace(strings.Split(contentType, ";")[0]))
	switch {
	case strings.Contains(baseContentType, "text/html"), strings.Contains(baseContentType, "application/xhtml+xml"):
		// Accepted as HTML.
	case baseContentType == "", strings.Contains(baseContentType, "application/octet-stream"):
		contentType = strings.ToLower(http.DetectContentType(initialChunk))
		baseContentType = contentType
	case strings.Contains(baseContentType, "text/plain"), strings.Contains(baseContentType, "application/xml"), strings.Contains(baseContentType, "text/xml"):
		if !isHTMLContentType(baseContentType, initialChunk) {
			return nil, "", fmt.Errorf("unsupported content type %q: only HTML webpages are extracted", rawContentType)
		}
	default:
		return nil, "", fmt.Errorf("unsupported content type %q: only HTML webpages are extracted", rawContentType)
	}
	if !isHTMLContentType(baseContentType, initialChunk) {
		return nil, "", fmt.Errorf("unsupported content type %q: only HTML webpages are extracted", rawContentType)
	}

	// Continue reading HTML only, with a one-byte sentinel to detect truncation.
	body := append([]byte(nil), initialChunk...)
	remaining := f.maxBodyBytes - int64(len(body))
	if remaining > 0 {
		rest, err := io.ReadAll(io.LimitReader(resp.Body, remaining+1))
		if err != nil {
			return nil, "", fmt.Errorf("failed reading response body: %w", err)
		}
		body = append(body, rest...)
	}
	if int64(len(body)) > f.maxBodyBytes {
		return nil, "", fmt.Errorf("response body exceeds %d bytes", f.maxBodyBytes)
	}

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
