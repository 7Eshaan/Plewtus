package extractor

import (
	"bytes"
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
	transport    http.RoundTripper
	timeout      time.Duration
	userAgent    string
	maxBodyBytes int64
	maxPDFBytes  int64
}

// FetcherOptions configures the Fetcher.
type FetcherOptions struct {
	Timeout      time.Duration
	MaxBodyBytes int64
	// MaxPDFBytes caps PDF downloads (default DefaultMaxPDFBytes).
	MaxPDFBytes int64
	UserAgent   string
	Transport   http.RoundTripper // Optional custom transport for testing
}

// Default limits
const (
	DefaultFetchTimeout = 5 * time.Second
	// Pages carrying large inline scripts and JSON state commonly exceed 2 MB
	// of HTML even when the article is short.
	DefaultMaxBodyBytes = 5 * 1024 * 1024 // 5 MB
	// Papers and reports are often larger than web pages.
	DefaultMaxPDFBytes = 15 * 1024 * 1024 // 15 MB
	// PDFContentType is returned by Fetch for PDF documents.
	PDFContentType   = "application/pdf"
	DefaultUserAgent = "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/128.0.0.0 Safari/537.36"
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

	maxPDF := opts.MaxPDFBytes
	if maxPDF <= 0 {
		maxPDF = DefaultMaxPDFBytes
	}

	ua := opts.UserAgent
	if ua == "" {
		ua = DefaultUserAgent
	}

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

	return &Fetcher{
		transport:    transport,
		timeout:      timeout,
		userAgent:    ua,
		maxBodyBytes: maxBytes,
		maxPDFBytes:  maxPDF,
	}
}

// newClient builds a per-fetch client. Each fetch gets its own cookie jar so
// cookies never leak between unrelated sites or accumulate in a long-running
// process, while redirect chains (e.g. consent hops) still keep their cookies.
func (f *Fetcher) newClient() *http.Client {
	jar, _ := cookiejar.New(nil)
	return &http.Client{
		Transport:     f.transport,
		CheckRedirect: SafeCheckRedirect(5),
		Jar:           jar,
		// No client-wide timeout: Fetch bounds web pages with f.timeout and
		// lets PDFs run until the caller's deadline.
	}
}

// Fetch safely downloads the HTML page or PDF at the given URL.
//
// Web pages must arrive within the fetcher's timeout. Once a response turns
// out to be a PDF, that limit is lifted and only ctx bounds the download:
// documents are larger than pages, and are worth the wait.
func (f *Fetcher) Fetch(ctx context.Context, targetURL string) ([]byte, string, error) {
	reqCtx, cancel := context.WithCancelCause(ctx)
	defer cancel(nil)
	pageTimer := time.AfterFunc(f.timeout, func() { cancel(errPageTimeout) })
	defer pageTimer.Stop()

	body, contentType, err := f.fetch(reqCtx, targetURL, func() { pageTimer.Stop() })
	if err != nil && ctx.Err() == nil && errors.Is(context.Cause(reqCtx), errPageTimeout) {
		return nil, "", fmt.Errorf("page did not load within %s: %w", f.timeout, context.DeadlineExceeded)
	}
	return body, contentType, err
}

// errPageTimeout marks a web page that exceeded the fetcher's timeout.
var errPageTimeout = errors.New("page fetch timeout")

// fetch performs Fetch; extendDeadline lifts the page timeout for PDFs.
func (f *Fetcher) fetch(ctx context.Context, targetURL string, extendDeadline func()) ([]byte, string, error) {
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
	resp, err := f.newClient().Do(req)
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
	// A PDF gets its longer time budget as soon as the headers or the first
	// bytes show it is one, not only once the whole prefix has arrived.
	declared := strings.ToLower(strings.TrimSpace(strings.Split(resp.Header.Get("Content-Type"), ";")[0]))
	if declared == "application/pdf" || declared == "application/x-pdf" {
		extendDeadline()
	}
	prefix := make([]byte, prefixLimit)
	n, readErr := readPrefix(resp.Body, prefix, func(got []byte) {
		if bytes.HasPrefix(bytes.TrimLeft(got, " \t\r\n"), []byte("%PDF-")) {
			extendDeadline()
		}
	})
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
	// PDFs are accepted when declared or sniffed and the file really is one.
	rawContentType := resp.Header.Get("Content-Type")
	contentType := strings.ToLower(strings.TrimSpace(rawContentType))
	baseContentType := strings.ToLower(strings.TrimSpace(strings.Split(contentType, ";")[0]))
	looksLikePDF := bytes.HasPrefix(bytes.TrimLeft(initialChunk, " \t\r\n"), []byte("%PDF-"))
	isPDF := false
	switch {
	case baseContentType == "application/pdf", baseContentType == "application/x-pdf":
		if !looksLikePDF {
			return nil, "", fmt.Errorf("content declared as %q is not a PDF file", rawContentType)
		}
		isPDF = true
	case strings.Contains(baseContentType, "text/html"), strings.Contains(baseContentType, "application/xhtml+xml"):
		// Accepted as HTML.
	case (baseContentType == "" || strings.Contains(baseContentType, "application/octet-stream")) && looksLikePDF:
		isPDF = true
	case baseContentType == "", strings.Contains(baseContentType, "application/octet-stream"):
		contentType = strings.ToLower(http.DetectContentType(initialChunk))
		baseContentType = contentType
	case strings.Contains(baseContentType, "text/plain"), strings.Contains(baseContentType, "application/xml"), strings.Contains(baseContentType, "text/xml"):
		if !isHTMLContentType(baseContentType, initialChunk) {
			return nil, "", fmt.Errorf("unsupported content type %q: only HTML webpages are extracted", rawContentType)
		}
	default:
		return nil, "", fmt.Errorf("unsupported content type %q: only HTML webpages and PDFs are extracted", rawContentType)
	}
	if !isPDF && !isHTMLContentType(baseContentType, initialChunk) {
		return nil, "", fmt.Errorf("unsupported content type %q: only HTML webpages and PDFs are extracted", rawContentType)
	}
	limit := f.maxBodyBytes
	if isPDF {
		limit = f.maxPDFBytes
		extendDeadline()
	}

	// Continue reading HTML or PDF only, with a one-byte sentinel to detect truncation.
	body := append([]byte(nil), initialChunk...)
	remaining := limit - int64(len(body))
	if remaining > 0 {
		rest, err := io.ReadAll(io.LimitReader(resp.Body, remaining+1))
		if err != nil {
			return nil, "", fmt.Errorf("failed reading response body: %w", err)
		}
		body = append(body, rest...)
	}
	if int64(len(body)) > limit {
		return nil, "", fmt.Errorf("response body exceeds %d bytes", limit)
	}

	if isPDF {
		return body, PDFContentType, nil
	}
	return toUTF8(body, rawContentType), contentType, nil
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

// readPrefix fills buf like io.ReadFull, calling inspect with the bytes
// received so far after every read.
func readPrefix(r io.Reader, buf []byte, inspect func([]byte)) (int, error) {
	n := 0
	for n < len(buf) {
		m, err := r.Read(buf[n:])
		n += m
		if m > 0 {
			inspect(buf[:n])
		}
		if err != nil {
			if errors.Is(err, io.EOF) && n > 0 {
				return n, io.ErrUnexpectedEOF
			}
			return n, err
		}
	}
	return n, nil
}
