package extractor

import (
	"context"
	"net/http"
	"net/http/httptest"
	"net/netip"
	"strings"
	"testing"

	"plewtus/internal/model"
)

var regressionBody = strings.Repeat("Concurrency in Go uses goroutines and channels to structure programs. ", 4)

func markdownFor(t *testing.T, html, pageURL string) string {
	t.Helper()
	p, err := ParseHTML([]byte(html))
	if err != nil {
		t.Fatal(err)
	}
	return ToMarkdownWithBase(p.MainNode, 4000, p.ResolveBase(pageURL))
}

// Regression: content inside layout wrappers must survive pruning.
func TestRegression_PruningKeepsContent(t *testing.T) {
	cases := map[string]string{
		"aspnet form wrapper": `<html><body><form id="aspnetForm"><div><h1>Docs</h1><p>` + regressionBody + `</p></div></form></body></html>`,
		"has-sidebar layout":  `<html><body><div class="layout has-sidebar"><main><h1>Guide</h1><p>` + regressionBody + `</p></main></div></body></html>`,
		"cookie-wrapped main": `<html><body><div class="cookie-consent-root"><main><p>` + regressionBody + `</p></main></div></body></html>`,
	}
	for name, html := range cases {
		if md := markdownFor(t, html, "https://example.com/"); !strings.Contains(md, "goroutines") {
			t.Errorf("%s: content was pruned, got %q", name, md)
		}
	}
}

func TestRegression_NoiseStillPruned(t *testing.T) {
	html := `<html><body><main><p>` + regressionBody + `</p><div class="cookie-banner">Accept all cookies now</div><div id="newsletter-signup">Subscribe today</div></main></body></html>`
	md := markdownFor(t, html, "https://example.com/")
	if strings.Contains(md, "Accept all cookies") || strings.Contains(md, "Subscribe today") {
		t.Fatalf("noise should be pruned, got %q", md)
	}
}

func TestRegression_ArticleHeaderTitleKept(t *testing.T) {
	html := `<html><body><header>Site nav</header><article><header><h1>The Real Title</h1></header><p>` + regressionBody + `</p></article></body></html>`
	md := markdownFor(t, html, "https://example.com/")
	if !strings.Contains(md, "# The Real Title") {
		t.Fatalf("article title lost: %q", md)
	}
	if strings.Contains(md, "Site nav") {
		t.Fatalf("page-level header should be pruned: %q", md)
	}
}

func TestRegression_InlineSpacingAndAbsoluteLinks(t *testing.T) {
	html := `<html><head><base href="/docs/"></head><body><article><p>` + regressionBody + `See <a href="install">install docs</a> and <strong>read</strong> the <a href="#top">top</a>.</p><img src="/img/a.png" alt="diagram"></article></body></html>`
	md := markdownFor(t, html, "https://example.com/guide")
	for _, want := range []string{
		"See [install docs](https://example.com/docs/install) and **read** the top.",
		"![diagram](https://example.com/img/a.png)",
	} {
		if !strings.Contains(md, want) {
			t.Errorf("missing %q in %q", want, md)
		}
	}
}

func TestRegression_NestedListsKeepItems(t *testing.T) {
	html := `<html><body><main><p>` + regressionBody + `</p><ul><li>One<ul><li>Child</li></ul></li><li>Two</li></ul></main></body></html>`
	md := markdownFor(t, html, "https://example.com/")
	if !strings.Contains(md, "* Two") || !strings.Contains(md, "Child") {
		t.Fatalf("list items lost: %q", md)
	}
}

// Regression: each mode returns a single representation (no duplicated payload).
func TestRegression_ModesDoNotDuplicatePayload(t *testing.T) {
	page := `<html><head><title>Go Guide</title></head><body><article><h1>Go</h1><p>` + regressionBody + `</p></article></body></html>`
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html")
		_, _ = w.Write([]byte(page))
	}))
	defer ts.Close()

	ext := New(ExtractorOptions{Fetcher: NewFetcher(FetcherOptions{Transport: ts.Client().Transport})})
	// The test server is on loopback, which ValidateURL blocks; exercise the
	// conversion path through a public-looking URL routed to the test server.
	ext.fetcher.transport = rewriteTransport{target: ts.URL, base: ts.Client().Transport}

	md, err := ext.Extract(context.Background(), "http://example.com/guide", &model.ExtractOptions{Mode: "markdown", MaxCharsPerPage: 4000})
	if err != nil {
		t.Fatal(err)
	}
	if md.Markdown == "" || md.Text != "" || md.Title != "Go Guide" {
		t.Fatalf("markdown mode should fill only markdown (and title): %+v", md)
	}
	hl, err := ext.Extract(context.Background(), "http://example.com/guide", &model.ExtractOptions{Mode: "highlights", Query: "goroutines", MaxCharsPerPage: 4000})
	if err != nil {
		t.Fatal(err)
	}
	if len(hl.Highlights) == 0 || hl.Text != "" {
		t.Fatalf("highlights mode should fill only highlights: %+v", hl)
	}
}

type rewriteTransport struct {
	target string
	base   http.RoundTripper
}

func (rt rewriteTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	clone := req.Clone(req.Context())
	u := *req.URL
	u.Scheme = "http"
	u.Host = strings.TrimPrefix(rt.target, "http://")
	clone.URL = &u
	return rt.base.RoundTrip(clone)
}

func TestRegression_FetcherHasNoSharedCookieJar(t *testing.T) {
	f := NewFetcher(FetcherOptions{})
	if f.newClient().Jar == f.newClient().Jar {
		t.Fatal("each fetch must get its own cookie jar")
	}
}

func TestRegression_TransitionRangesBlocked(t *testing.T) {
	for _, raw := range []string{"64:ff9b::7f00:1", "2002:7f00:1::1", "2001:0:4136:e378::1", "100::1"} {
		if !IsBlockedAddr(netip.MustParseAddr(raw)) {
			t.Errorf("%s should be blocked", raw)
		}
	}
	if IsBlockedAddr(netip.MustParseAddr("2606:4700::1111")) {
		t.Error("public IPv6 must not be blocked")
	}
}

// Regression: truncating inside a code block must not leave the fence open.
func TestRegression_TruncationClosesCodeFence(t *testing.T) {
	page := "<html><body><article><h1>Title</h1><p>" + regressionBody + "</p><pre><code>" + strings.Repeat("line of code here\n", 60) + "</code></pre><p>after</p></article></body></html>"
	p, err := ParseHTML([]byte(page))
	if err != nil {
		t.Fatal(err)
	}
	md := ToMarkdownWithBase(p.MainNode, len(regressionBody)+300, nil)
	if n := strings.Count(md, "```"); n == 0 || n%2 != 0 {
		t.Fatalf("expected balanced code fences after truncation, got %d", n)
	}
}

// Regression: a failed page extraction is reported on the result instead of
// silently leaving it without content.
func TestRegression_BatchReportsPerResultErrors(t *testing.T) {
	ext := New(ExtractorOptions{})
	results := ext.ExtractBatch(context.Background(),
		[]model.SearchResult{{URL: "http://127.0.0.1/private"}},
		&model.ExtractOptions{Enabled: true, Mode: "markdown", MaxPages: 1})
	if results[0].Content != nil || !strings.Contains(results[0].ContentError, "blocked") {
		t.Fatalf("expected a content_error explaining the block, got %+v", results[0])
	}
}
