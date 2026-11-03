package extractor

import (
	"bytes"
	"context"
	"io"
	"net/http"
	"strings"
	"testing"
	"time"

	"plewtus/internal/model"
)

const sampleHTML = `<!DOCTYPE html>
<html lang="en">
<head>
    <meta charset="UTF-8">
    <title>Go Concurrency Guide - TechBlog</title>
    <meta name="description" content="A comprehensive tutorial on Go goroutines and channels.">
    <meta name="author" content="Jane Doe">
    <meta property="article:published_time" content="2026-05-15T10:00:00Z">
</head>
<body>
    <header>
        <nav><a href="/">Home</a> | <a href="/blog">Blog</a></nav>
        <div class="banner-ad">Banner Ad Content</div>
    </header>

    <div class="cookie-banner">Please accept all cookies.</div>

    <main>
        <article>
            <h1>Understanding Go Concurrency</h1>
            <p>Go provides first-class support for concurrent programming through <strong>goroutines</strong> and <em>channels</em>.</p>
            <p>Goroutines are extremely lightweight threads managed directly by the Go runtime rather than operating system threads.</p>

            <h2>Key Features</h2>
            <ul>
                <li>Lightweight thread creation with small initial stack sizes.</li>
                <li>Channels provide safe communication between concurrent tasks.</li>
                <li>The select statement enables multiplexing on channel operations.</li>
            </ul>

            <h2>Example Code</h2>
            <pre><code>func worker(ch chan int) {
    ch &lt;- 42
}</code></pre>

            <blockquote>Concurrency is not parallelism, but it enables parallelism when multiple execution units exist.</blockquote>

            <p>For more details, visit <a href="https://go.dev/doc/">official Go documentation</a>.</p>
        </article>
    </main>

    <aside class="sidebar-widget">
        <h3>Sponsored Links</h3>
        <p>Buy our cloud product today.</p>
    </aside>

    <footer>
        <p>&copy; 2026 TechBlog. All rights reserved.</p>
    </footer>

    <script>console.log("analytics tracking");</script>
</body>
</html>`

func TestParseHTML_MetadataAndPruning(t *testing.T) {
	parsed, err := ParseHTML([]byte(sampleHTML))
	if err != nil {
		t.Fatalf("ParseHTML failed: %v", err)
	}

	// Verify metadata
	if parsed.Metadata.Title != "Go Concurrency Guide - TechBlog" {
		t.Errorf("expected title 'Go Concurrency Guide - TechBlog', got %q", parsed.Metadata.Title)
	}
	if parsed.Metadata.Description != "A comprehensive tutorial on Go goroutines and channels." {
		t.Errorf("expected description, got %q", parsed.Metadata.Description)
	}
	if parsed.Metadata.Author != "Jane Doe" {
		t.Errorf("expected author 'Jane Doe', got %q", parsed.Metadata.Author)
	}
	if parsed.Metadata.PublishedDate != "2026-05-15T10:00:00Z" {
		t.Errorf("expected published date, got %q", parsed.Metadata.PublishedDate)
	}

	// Verify noise pruning: script, nav, cookie banner, sponsored sidebar should be pruned
	text := ToText(parsed.MainNode, 0)
	if strings.Contains(text, "analytics tracking") {
		t.Errorf("expected script to be pruned")
	}
	if strings.Contains(text, "Please accept all cookies") {
		t.Errorf("expected cookie banner to be pruned")
	}
	if strings.Contains(text, "Banner Ad Content") {
		t.Errorf("expected banner ad to be pruned")
	}
	if strings.Contains(text, "Sponsored Links") {
		t.Errorf("expected sponsored sidebar to be pruned")
	}

	// Verify main article content is preserved
	if !strings.Contains(text, "Understanding Go Concurrency") {
		t.Errorf("expected main heading to be present")
	}
	if !strings.Contains(text, "goroutines and channels") {
		t.Errorf("expected main body paragraph to be present")
	}
}

func TestToMarkdown(t *testing.T) {
	parsed, err := ParseHTML([]byte(sampleHTML))
	if err != nil {
		t.Fatalf("ParseHTML failed: %v", err)
	}

	md := ToMarkdown(parsed.MainNode, 0)

	// Check headings
	if !strings.Contains(md, "# Understanding Go Concurrency") {
		t.Errorf("expected markdown H1, got:\n%s", md)
	}
	if !strings.Contains(md, "## Key Features") {
		t.Errorf("expected markdown H2, got:\n%s", md)
	}

	// Check bold and italics
	if !strings.Contains(md, "**goroutines**") {
		t.Errorf("expected bold **goroutines**, got:\n%s", md)
	}
	if !strings.Contains(md, "*channels*") {
		t.Errorf("expected italics *channels*, got:\n%s", md)
	}

	// Check list items
	if !strings.Contains(md, "* Lightweight thread creation") {
		t.Errorf("expected bullet point, got:\n%s", md)
	}

	// Check code block
	if !strings.Contains(md, "```\nfunc worker(ch chan int)") {
		t.Errorf("expected fenced code block, got:\n%s", md)
	}

	// Check blockquote
	if !strings.Contains(md, "> Concurrency is not parallelism") {
		t.Errorf("expected blockquote, got:\n%s", md)
	}

	// Check link
	if !strings.Contains(md, "[official Go documentation](https://go.dev/doc/)") {
		t.Errorf("expected markdown link, got:\n%s", md)
	}
}

func TestToMarkdown_Table(t *testing.T) {
	tableHTML := `<table>
		<tr><th>Engine</th><th>Status</th></tr>
		<tr><td>DuckDuckGo</td><td>Working</td></tr>
		<tr><td>Bing</td><td>Working</td></tr>
	</table>`

	parsed, err := ParseHTML([]byte(tableHTML))
	if err != nil {
		t.Fatalf("ParseHTML failed: %v", err)
	}

	md := ToMarkdown(parsed.MainNode, 0)
	if !strings.Contains(md, "| Engine | Status |") {
		t.Errorf("expected table header in markdown, got:\n%s", md)
	}
	if !strings.Contains(md, "| DuckDuckGo | Working |") {
		t.Errorf("expected table row in markdown, got:\n%s", md)
	}
}

func TestExtractHighlights(t *testing.T) {
	text := `Go provides first-class support for concurrent programming through goroutines and channels.
Goroutines are extremely lightweight threads managed directly by the Go runtime rather than operating system threads.
Channels provide safe communication between concurrent tasks.
All rights reserved by TechBlog 2026.
Please sign up for free updates.
The select statement enables multiplexing on channel operations without blocking.`

	highlights := ExtractHighlights(text, "goroutines channels", 3)
	if len(highlights) == 0 {
		t.Fatalf("expected highlights, got none")
	}

	if len(highlights) > 3 {
		t.Errorf("expected at most 3 highlights, got %d", len(highlights))
	}

	for _, h := range highlights {
		if strings.Contains(h, "All rights reserved") || strings.Contains(h, "sign up for free") {
			t.Errorf("highlight contains boilerplate: %q", h)
		}
	}

	// First highlight should be high scoring sentence matching query terms
	if !strings.Contains(highlights[0], "goroutines") {
		t.Errorf("expected top highlight to mention goroutines, got %q", highlights[0])
	}
}

func TestTruncateCleanly(t *testing.T) {
	long := "Sentence one is clear. Sentence two is very informative. Sentence three is also great."
	truncated := truncateCleanly(long, 35)

	// Should truncate after "Sentence one is clear."
	if truncated != "Sentence one is clear." {
		t.Errorf("expected sentence boundary truncation 'Sentence one is clear.', got %q", truncated)
	}

	// Word boundary fallback
	noPunct := "Alpha beta gamma delta epsilon zeta eta theta iota kappa"
	wordTrunc := truncateCleanly(noPunct, 20)
	if !strings.HasSuffix(wordTrunc, " …") {
		t.Errorf("expected ellipsis suffix, got %q", wordTrunc)
	}
	if len(wordTrunc) > 25 {
		t.Errorf("truncated string too long: %q", wordTrunc)
	}
}

// mockRoundTripper allows unit testing of Fetcher without network access.
type mockRoundTripper struct {
	roundTripFunc func(req *http.Request) (*http.Response, error)
}

func (m *mockRoundTripper) RoundTrip(req *http.Request) (*http.Response, error) {
	return m.roundTripFunc(req)
}

func TestFetcher_MIMETypeGuard(t *testing.T) {
	mockTransport := &mockRoundTripper{
		roundTripFunc: func(req *http.Request) (*http.Response, error) {
			if strings.Contains(req.URL.Path, "doc.pdf") {
				return &http.Response{
					StatusCode: http.StatusOK,
					Status:     "200 OK",
					Header: http.Header{
						"Content-Type": []string{"application/pdf"},
					},
					Body: io.NopCloser(bytes.NewReader([]byte("%PDF-1.4 binary content"))),
				}, nil
			}

			if strings.Contains(req.URL.Path, "page.html") {
				return &http.Response{
					StatusCode: http.StatusOK,
					Status:     "200 OK",
					Header: http.Header{
						"Content-Type": []string{"text/html; charset=utf-8"},
					},
					Body: io.NopCloser(bytes.NewReader([]byte(sampleHTML))),
				}, nil
			}

			return &http.Response{
				StatusCode: http.StatusNotFound,
				Status:     "404 Not Found",
				Body:       io.NopCloser(strings.NewReader("not found")),
			}, nil
		},
	}

	fetcher := NewFetcher(FetcherOptions{
		Transport: mockTransport,
	})

	ctx := context.Background()

	// 1. PDF should be rejected by Content-Type guard
	_, _, err := fetcher.Fetch(ctx, "https://example.com/doc.pdf")
	if err == nil {
		t.Errorf("expected error when fetching PDF, got nil")
	}
	if !strings.Contains(err.Error(), "unsupported content type") {
		t.Errorf("expected unsupported content type error, got: %v", err)
	}

	// 2. HTML should be accepted
	body, contentType, err := fetcher.Fetch(ctx, "https://example.com/page.html")
	if err != nil {
		t.Fatalf("unexpected error fetching HTML: %v", err)
	}
	if len(body) == 0 {
		t.Errorf("expected non-empty body")
	}
	if !strings.Contains(contentType, "text/html") {
		t.Errorf("expected text/html content type, got %q", contentType)
	}
}

func TestExtractor_ExtractBatch(t *testing.T) {
	mockTransport := &mockRoundTripper{
		roundTripFunc: func(req *http.Request) (*http.Response, error) {
			if strings.Contains(req.URL.Path, "error") {
				return &http.Response{
					StatusCode: http.StatusInternalServerError,
					Status:     "500 Internal Server Error",
					Body:       io.NopCloser(strings.NewReader("server error")),
				}, nil
			}

			return &http.Response{
				StatusCode: http.StatusOK,
				Status:     "200 OK",
				Header: http.Header{
					"Content-Type": []string{"text/html"},
				},
				Body: io.NopCloser(bytes.NewReader([]byte(sampleHTML))),
			}, nil
		},
	}

	fetcher := NewFetcher(FetcherOptions{
		Transport: mockTransport,
	})

	ext := New(ExtractorOptions{
		Fetcher: fetcher,
		Timeout: 2 * time.Second,
	})

	results := []model.SearchResult{
		{ID: "1", Title: "Result 1", URL: "https://example.com/1"},
		{ID: "2", Title: "Result 2", URL: "https://example.com/error"},
		{ID: "3", Title: "Result 3", URL: "https://example.com/3"},
		{ID: "4", Title: "Result 4", URL: "https://example.com/4"},
		{ID: "5", Title: "Result 5", URL: "https://example.com/5"},
	}

	// Test 1: Extraction disabled -> results unchanged
	unchanged := ext.ExtractBatch(context.Background(), results, &model.ExtractOptions{Enabled: false})
	for _, r := range unchanged {
		if r.Content != nil {
			t.Errorf("expected content nil when extract disabled")
		}
	}

	// Test 2: MaxPages = 2 -> only first 2 processed, errors handled gracefully
	opts := &model.ExtractOptions{
		Enabled:         true,
		Mode:            "markdown",
		MaxPages:        2,
		MaxCharsPerPage: 1000,
	}

	extracted := ext.ExtractBatch(context.Background(), results, opts)

	// Result 0: succeeded
	if extracted[0].Content == nil {
		t.Errorf("expected content for result 0, got nil")
	} else {
		if !strings.Contains(extracted[0].Content.Markdown, "# Understanding Go Concurrency") {
			t.Errorf("expected markdown content, got:\n%s", extracted[0].Content.Markdown)
		}
		if extracted[0].Content.Author != "Jane Doe" {
			t.Errorf("expected author Jane Doe, got %q", extracted[0].Content.Author)
		}
	}

	// Result 1: failed (500 error) -> Content remains nil without erroring
	if extracted[1].Content != nil {
		t.Errorf("expected content nil for failed result 1")
	}

	// Results 2, 3, 4: beyond MaxPages (2) -> Content must be nil
	for i := 2; i < len(extracted); i++ {
		if extracted[i].Content != nil {
			t.Errorf("result %d exceeded MaxPages limit of 2, but content was extracted", i)
		}
	}
}
