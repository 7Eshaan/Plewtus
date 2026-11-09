package provider

import (
	"context"
	"crypto/tls"
	"fmt"
	"io"
	"net/http"
	"net/http/cookiejar"
	"time"
)

// HTTPClient wraps an http.Client with polite scraping defaults.
type HTTPClient struct {
	transport *http.Transport
	timeout   time.Duration
	userAgent string
}

// ClientOption customizes an HTTPClient.
type ClientOption func(*http.Transport)

// WithHTTP1Only disables HTTP/2 negotiation. Some search engines (notably
// Yahoo) terminate HTTP/2 connections from non-browser TLS stacks with an
// unexpected EOF while serving HTTP/1.1 normally.
func WithHTTP1Only() ClientOption {
	return func(t *http.Transport) {
		t.ForceAttemptHTTP2 = false
		t.TLSNextProto = map[string]func(string, *tls.Conn) http.RoundTripper{}
		if t.TLSClientConfig == nil {
			t.TLSClientConfig = &tls.Config{}
		}
		t.TLSClientConfig.NextProtos = []string{"http/1.1"}
	}
}

// NewHTTPClient creates a client configured with transport connection pooling and isolated cookie jars.
func NewHTTPClient(timeout time.Duration, userAgent string, opts ...ClientOption) *HTTPClient {
	transport := &http.Transport{Proxy: http.ProxyFromEnvironment}
	if defTr, ok := http.DefaultTransport.(*http.Transport); ok {
		transport = defTr.Clone()
	}
	for _, opt := range opts {
		opt(transport)
	}

	return &HTTPClient{
		transport: transport,
		timeout:   timeout,
		userAgent: userAgent,
	}
}

// Do executes an HTTP request with isolated cookies, setting polite default headers.
func (c *HTTPClient) Do(ctx context.Context, req *http.Request) (*http.Response, error) {
	req = req.WithContext(ctx)

	if c.userAgent != "" && req.Header.Get("User-Agent") == "" {
		req.Header.Set("User-Agent", c.userAgent)
	}
	if req.Header.Get("Accept") == "" {
		req.Header.Set("Accept", "text/html,application/xhtml+xml,application/xml;q=0.9,image/avif,image/webp,*/*;q=0.8")
	}
	if req.Header.Get("Accept-Language") == "" {
		req.Header.Set("Accept-Language", "en-US,en;q=0.9")
	}

	// Use an isolated cookie jar per request so search engines cannot track
	// scraping sessions across different queries, while still supporting redirects within this call.
	jar, _ := cookiejar.New(nil)
	client := &http.Client{
		Transport: c.transport,
		Jar:       jar,
		Timeout:   c.timeout,
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			if len(via) >= 10 {
				return fmt.Errorf("stopped after 10 redirects")
			}
			if c.userAgent != "" {
				req.Header.Set("User-Agent", c.userAgent)
			}
			return nil
		},
	}

	return client.Do(req)
}

// CloseIdleConnections closes any idle TCP connections in the client transport.
func (c *HTTPClient) CloseIdleConnections() {
	if c.transport != nil {
		c.transport.CloseIdleConnections()
	}
}

// ReadBody reads and closes the response body.
func ReadBody(resp *http.Response) ([]byte, error) {
	defer resp.Body.Close()
	const maxSearchBodyBytes = 4 << 20
	body, err := io.ReadAll(io.LimitReader(resp.Body, maxSearchBodyBytes+1))
	if err != nil {
		return nil, err
	}
	if len(body) > maxSearchBodyBytes {
		return nil, fmt.Errorf("search response body exceeds %d bytes", maxSearchBodyBytes)
	}
	return body, nil
}
