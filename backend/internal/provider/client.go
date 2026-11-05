package provider

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"net/http/cookiejar"
	"time"
)

// HTTPClient wraps an http.Client with polite scraping defaults.
type HTTPClient struct {
	client    *http.Client
	userAgent string
}

// NewHTTPClient creates a client configured with cookie jar and redirect handling.
func NewHTTPClient(timeout time.Duration, userAgent string) *HTTPClient {
	jar, _ := cookiejar.New(nil)
	var transport *http.Transport
	if defTr, ok := http.DefaultTransport.(*http.Transport); ok {
		transport = defTr.Clone()
	}

	return &HTTPClient{
		client: &http.Client{
			Transport: transport,
			Jar:       jar,
			Timeout:   timeout,
			CheckRedirect: func(req *http.Request, via []*http.Request) error {
				if len(via) >= 10 {
					return fmt.Errorf("stopped after 10 redirects")
				}
				// Copy User-Agent to redirected requests
				if userAgent != "" {
					req.Header.Set("User-Agent", userAgent)
				}
				return nil
			},
		},
		userAgent: userAgent,
	}
}

// Do executes an HTTP request, setting polite default headers.
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

	return c.client.Do(req)
}

// CloseIdleConnections closes any idle TCP connections in the client transport.
func (c *HTTPClient) CloseIdleConnections() {
	if tr, ok := c.client.Transport.(*http.Transport); ok && tr != nil {
		tr.CloseIdleConnections()
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
