package extractor

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"plewtus/internal/model"
)

var articleProse = strings.Repeat("Rising temperatures shorten growing seasons and reduce crop yields across many regions. ", 6)

// languageList mimics Wikipedia's interlanguage picker: a long list of links.
func languageList(class string) string {
	var b strings.Builder
	b.WriteString(`<div class="` + class + `"><ul>`)
	for _, lang := range []string{"Deutsch", "Español", "Français", "Italiano", "日本語", "Português", "Русский", "中文"} {
		b.WriteString(`<li><a href="https://xx.wikipedia.org/wiki/Page">` + lang + `</a></li>`)
	}
	b.WriteString(`</ul></div>`)
	return b.String()
}

// serveExtractor returns an extractor whose fetches of http://example.com/
// are answered by handler.
func serveExtractor(t *testing.T, handler http.HandlerFunc) *Extractor {
	t.Helper()
	ts := httptest.NewServer(handler)
	t.Cleanup(ts.Close)
	ext := New(ExtractorOptions{Fetcher: NewFetcher(FetcherOptions{Transport: ts.Client().Transport})})
	ext.fetcher.transport = rewriteTransport{target: ts.URL, base: ts.Client().Transport}
	return ext
}

func htmlHandler(page, contentType string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", contentType)
		_, _ = w.Write([]byte(page))
	}
}

func TestQuality_LinkListsAreNotContent(t *testing.T) {
	// The language list sits inside <main> ahead of the article, so before the
	// fix it filled the whole character budget.
	page := `<html><body><main id="content"><h1>Effects of climate change</h1>` + languageList("lang-list") +
		`<div id="body"><p>` + articleProse + `</p></div></main></body></html>`
	md := markdownFor(t, page, "https://en.wikipedia.org/wiki/X")
	if strings.Contains(md, "Deutsch") || strings.Contains(md, "日本語") {
		t.Fatalf("language list should be dropped: %q", md)
	}
	if !strings.Contains(md, "growing seasons") {
		t.Fatalf("article lost: %q", md)
	}
}

func TestQuality_NavigationChromePruned(t *testing.T) {
	page := `<html><body>
		<div class="vector-menu-content">Main menu Contents Current events Random article About</div>
		<div role="navigation">Courses Tutorials Practice Jobs</div>
		<div aria-hidden="true">Decorative duplicate heading</div>
		<template><div class="card">${hit.model} ${hit.price}</div></template>
		<article><p>` + articleProse + `</p></article></body></html>`
	md := markdownFor(t, page, "https://example.com/")
	for _, noise := range []string{"Current events", "Tutorials", "Decorative duplicate", "${hit.model}"} {
		if strings.Contains(md, noise) {
			t.Errorf("noise %q should be pruned: %q", noise, md)
		}
	}
	if !strings.Contains(md, "growing seasons") {
		t.Fatalf("article lost: %q", md)
	}
}

func TestQuality_PrefersArticleOverChromeWrapper(t *testing.T) {
	// GitHub-style: <main> wraps repository chrome (with some prose of its
	// own) and the README <article>; the README should be what is returned.
	page := `<html><body><main>
		<div class="repo-header"><p>You must be signed in to change notification settings for this public repository.</p>
		<p>Stars 3.1k, Watchers 29, Forks 266, Branches and Tags are available for this repository.</p></div>
		<article class="markdown-body"><h1>Concurrency patterns</h1><p>` + articleProse + `</p><p>` + articleProse + `</p></article>
		</main></body></html>`
	md := markdownFor(t, page, "https://github.com/x/y")
	if strings.Contains(md, "signed in") {
		t.Fatalf("repository chrome should not be part of the content: %q", md)
	}
	if !strings.HasPrefix(md, "# Concurrency patterns") {
		t.Fatalf("expected the README article, got %q", md)
	}
}

func TestQuality_FindsContentWithoutSemanticTags(t *testing.T) {
	// No <main>/<article>: the container holding the paragraphs must win over
	// <body>, which also holds the site's own prose (about box, legal text).
	page := `<html><body>
		<div class="top">Welcome to our wonderful website where we write about many interesting topics every day.</div>
		<div class="post-body"><h2>Impacts</h2><p>` + articleProse + `</p><p>` + articleProse + `</p><p>` + articleProse + `</p></div>
		<div class="legal">This site is operated by Example Media Group and all content belongs to its respective authors.</div>
		</body></html>`
	md := markdownFor(t, page, "https://example.com/")
	if strings.Contains(md, "Welcome to our wonderful") || strings.Contains(md, "Example Media Group") {
		t.Fatalf("site chrome should be excluded: %q", md)
	}
	if !strings.Contains(md, "## Impacts") {
		t.Fatalf("content container lost: %q", md)
	}
}

func TestQuality_ShortContentListsAndTablesSurvive(t *testing.T) {
	// Link density alone must not delete ordinary lists and data tables.
	page := `<html><body><article><p>` + articleProse + `</p>
		<ul><li>Wheat yields fall by about six percent per degree of warming.</li><li>Rice yields fall by about three percent.</li></ul>
		<table><tr><th>Crop</th><th>Change</th></tr><tr><td>Maize</td><td>-7%</td></tr></table>
		<p>See the <a href="/report">full report</a> for methods.</p></article></body></html>`
	md := markdownFor(t, page, "https://example.com/")
	for _, want := range []string{"Wheat yields", "| Maize | -7% |", "[full report](https://example.com/report)"} {
		if !strings.Contains(md, want) {
			t.Errorf("missing %q in %q", want, md)
		}
	}
}

func TestQuality_JavaScriptShellFallsBackToDescription(t *testing.T) {
	shell := `<html><head><title>Video - YouTube</title><meta name="description" content="A walkthrough of how heat pumps move heat instead of generating it."></head>
		<body><div id="app"></div><div class="links"><a href="/about">About</a><a href="/press">Press</a><a href="/terms">Terms</a></div></body></html>`
	ext := serveExtractor(t, htmlHandler(shell, "text/html"))
	content, err := ext.Extract(context.Background(), "http://example.com/watch", &model.ExtractOptions{Mode: "markdown", MaxCharsPerPage: 4000})
	if err != nil {
		t.Fatal(err)
	}
	if content.Markdown != "A walkthrough of how heat pumps move heat instead of generating it." {
		t.Fatalf("expected the page description, got %q", content.Markdown)
	}
}

func TestQuality_JavaScriptShellWithoutDescriptionIsAnError(t *testing.T) {
	shell := `<html><head><title>App</title></head><body><div id="root"></div><noscript>You need to enable JavaScript to run this app.</noscript></body></html>`
	ext := serveExtractor(t, htmlHandler(shell, "text/html"))
	_, err := ext.Extract(context.Background(), "http://example.com/app", &model.ExtractOptions{Mode: "text", MaxCharsPerPage: 4000})
	if !errors.Is(err, ErrNoReadableContent) {
		t.Fatalf("expected ErrNoReadableContent, got %v", err)
	}

	results := ext.ExtractBatch(context.Background(), []model.SearchResult{{URL: "http://example.com/app"}},
		&model.ExtractOptions{Enabled: true, Mode: "markdown", MaxPages: 1, MaxCharsPerPage: 4000})
	if results[0].Content != nil || !strings.Contains(results[0].ContentError, "JavaScript") {
		t.Fatalf("batch should report the unreadable page: %+v", results[0])
	}
}

func TestQuality_MarkdownLeadsWithTitleWhenHeadingMissing(t *testing.T) {
	page := `<html><head><title>Effects of climate change on agriculture</title></head><body><div class="post"><p>` +
		articleProse + `</p><p>` + articleProse + `</p></div></body></html>`
	ext := serveExtractor(t, htmlHandler(page, "text/html"))
	content, err := ext.Extract(context.Background(), "http://example.com/p", &model.ExtractOptions{Mode: "markdown", MaxCharsPerPage: 300})
	if err != nil {
		t.Fatal(err)
	}
	if !strings.HasPrefix(content.Markdown, "# Effects of climate change on agriculture\n\n") {
		t.Fatalf("expected title heading, got %q", content.Markdown)
	}
	if n := len([]rune(content.Markdown)); n > 300 {
		t.Fatalf("title must fit within the character budget, got %d chars", n)
	}
}

func TestQuality_Latin1PagesAreDecoded(t *testing.T) {
	// "Café crème – naïve résumé" encoded as Windows-1252.
	latin1 := "<html><head><meta charset=\"iso-8859-1\"></head><body><article><p>Caf\xe9 cr\xe8me \x96 na\xefve r\xe9sum\xe9. " +
		articleProse + "</p></article></body></html>"
	ext := serveExtractor(t, htmlHandler(latin1, "text/html"))
	content, err := ext.Extract(context.Background(), "http://example.com/fr", &model.ExtractOptions{Mode: "text", MaxCharsPerPage: 4000})
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(content.Text, "Café crème – naïve résumé") {
		t.Fatalf("latin-1 text was not decoded: %q", content.Text[:60])
	}
}

func TestQuality_UTF8PagesAreUntouched(t *testing.T) {
	body := []byte("<meta charset=\"iso-8859-1\"><p>Café</p>") // mislabelled but valid UTF-8
	if got := toUTF8(body, "text/html"); string(got) != string(body) {
		t.Fatalf("valid UTF-8 must not be re-decoded: %q", got)
	}
}

func TestQuality_HighlightsKeepHeadingsSeparate(t *testing.T) {
	text := "Quandale Dingle\n\nUpdated Jan 30, 2026 by Zach\n\n" +
		"Quandale Dingle is the name of a real high school football player whose name became a meme in 2022.\n\n" +
		"The meme spread on TikTok, where users made voice-over videos about him."
	highlights := ExtractHighlights(text, "Who is quandale dingle", 3)
	if len(highlights) == 0 {
		t.Fatal("expected highlights")
	}
	for _, h := range highlights {
		if strings.HasPrefix(h, "Quandale Dingle Updated") || strings.Contains(h, "Dingle\n") {
			t.Errorf("heading glued to the next line: %q", h)
		}
	}
	if !strings.Contains(highlights[0], "real high school football player") {
		t.Errorf("expected the defining sentence first, got %q", highlights[0])
	}
}

func TestQuality_HighlightTermsDropStopWords(t *testing.T) {
	got := strings.Join(highlightTerms("Who is the best C++ compiler, and how does it work?"), ",")
	if got != "c++,compiler,work" {
		t.Fatalf("unexpected terms %q", got)
	}
}

func TestQuality_StreamedHiddenContentIsKept(t *testing.T) {
	// Next.js streaming SSR sends the article in <div hidden> and swaps it in
	// with JavaScript; treating `hidden` as noise emptied these pages.
	page := `<html><body><div id="__next"><template id="B:0"></template><nav>Home Docs</nav></div>
		<div hidden id="S:0"><h1>Reciprocal Rank Fusion</h1><p>` + articleProse + `</p></div></body></html>`
	md := markdownFor(t, page, "https://example.com/")
	if !strings.Contains(md, "growing seasons") {
		t.Fatalf("streamed content was dropped: %q", md)
	}
}

func TestQuality_NoisyNamedWrapperHoldingThePageIsKept(t *testing.T) {
	// LogRocket wraps its whole post in <div class="sidebar-container">.
	page := `<html><body><header>Site</header><div class="sidebar-container"><div class="post"><h1>useEffect cleanup</h1><p>` +
		articleProse + `</p><p>` + articleProse + `</p></div><div class="recent"><p>Recent posts: a short teaser.</p></div></div></body></html>`
	md := markdownFor(t, page, "https://example.com/")
	if !strings.Contains(md, "# useEffect cleanup") || !strings.Contains(md, "growing seasons") {
		t.Fatalf("post inside a noisy-named layout wrapper was dropped: %q", md)
	}
}

func TestQuality_LinkHeavyPageIsNotEmptied(t *testing.T) {
	// A listing page (product picks, all links) is still the page's content.
	var b strings.Builder
	b.WriteString(`<html><body><div class="page">`)
	for i := 0; i < 12; i++ {
		b.WriteString(`<div class="pick"><a href="/p">Budget Laptop Model Number Twelve Pro Plus</a> <span>$499</span></div>`)
	}
	b.WriteString(`</div></body></html>`)
	md := markdownFor(t, b.String(), "https://example.com/")
	if !strings.Contains(md, "Budget Laptop Model") {
		t.Fatalf("listing page was emptied: %q", md)
	}
}

func TestQuality_LinkCollectionPagesKeepTheirLinks(t *testing.T) {
	// go.dev/wiki/LearnConcurrency: the article is a reading list. Removing
	// link lists would leave nothing, so they must be kept on such pages.
	var b strings.Builder
	b.WriteString(`<html><body><main><article><h1>Learn Concurrency</h1>`)
	for section := 0; section < 3; section++ {
		b.WriteString(`<h2>Section</h2><ul>`)
		for i := 0; i < 5; i++ {
			b.WriteString(`<li><a href="/talk">Go Concurrency Patterns talk by Rob Pike</a></li>`)
		}
		b.WriteString(`</ul>`)
	}
	b.WriteString(`</article></main></body></html>`)
	md := markdownFor(t, b.String(), "https://go.dev/wiki/LearnConcurrency")
	if !strings.Contains(md, "[Go Concurrency Patterns talk by Rob Pike](https://go.dev/talk)") {
		t.Fatalf("reading list was dropped: %q", md)
	}
}
