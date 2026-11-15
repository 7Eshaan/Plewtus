package extractor

import (
	"context"
	"fmt"
	"strings"
	"testing"
	"unicode/utf8"

	"plewtus/internal/model"
)

// longGuide builds a long article whose answer sits in section 7 of 9.
func longGuide() string {
	var b strings.Builder
	b.WriteString(`<html><head><title>Hybrid search guide</title></head><body><article><h1>Hybrid search guide</h1>`)
	b.WriteString(`<p>This guide explains how search systems combine keyword and vector retrieval into a single result list for applications.</p>`)
	for i := 1; i <= 9; i++ {
		b.WriteString(fmt.Sprintf(`<h2>Section %d</h2>`, i))
		if i == 7 {
			b.WriteString(`<h3>Reciprocal rank fusion</h3><p>Reciprocal rank fusion scores each document by summing one over k plus its rank in every list; the constant k is usually set to 60 so that lower ranks still contribute.</p>`)
			b.WriteString("<pre><code>score = sum(1 / (60 + rank))\nfor each list</code></pre>")
			continue
		}
		for j := 0; j < 3; j++ {
			b.WriteString(fmt.Sprintf(`<p>Paragraph %d.%d discusses indexing pipelines, tokenization, storage layouts, sharding, replication, and the operational trade-offs of running large clusters in production environments.</p>`, i, j))
		}
	}
	b.WriteString(`</article></body></html>`)
	return b.String()
}

func passagesFor(t *testing.T, page, query string, budget int, markdown bool) string {
	t.Helper()
	p, err := ParseHTML([]byte(page))
	if err != nil {
		t.Fatal(err)
	}
	return selectPassages(p.MainNode, p.ResolveBase("https://example.com/guide"), query, budget, markdown)
}

func TestPassages_FindTheAnswerDeepInThePage(t *testing.T) {
	out := passagesFor(t, longGuide(), "reciprocal rank fusion k constant", 1500, true)
	if out == "" {
		t.Fatal("expected passages")
	}
	if n := utf8.RuneCountInString(out); n > 1500 {
		t.Fatalf("budget exceeded: %d chars", n)
	}
	for _, want := range []string{
		"This guide explains how search systems",     // the intro is always kept
		"## Section 7", "### Reciprocal rank fusion", // the heading path of the match
		"usually set to 60",
		GapMarker,
	} {
		if !strings.Contains(out, want) {
			t.Errorf("missing %q in:\n%s", want, out)
		}
	}
	if strings.Count(out, "```")%2 != 0 {
		t.Errorf("code block was cut:\n%s", out)
	}
	// The match comes after the intro, in page order.
	if strings.Index(out, "This guide explains") > strings.Index(out, "usually set to 60") {
		t.Errorf("passages out of page order:\n%s", out)
	}
}

func TestPassages_TextModeUsesHeadingPaths(t *testing.T) {
	out := passagesFor(t, longGuide(), "reciprocal rank fusion k constant", 1500, false)
	if !strings.Contains(out, "Section 7 › Reciprocal rank fusion") || strings.Contains(out, "## ") {
		t.Fatalf("expected a plain-text heading path, got:\n%s", out)
	}
}

func TestPassages_NoQueryOrNoMatchFallsBack(t *testing.T) {
	if out := passagesFor(t, longGuide(), "", 1500, true); out != "" {
		t.Errorf("no query should fall back to the lead, got %q", out)
	}
	if out := passagesFor(t, longGuide(), "kubernetes helm charts", 1500, true); out != "" {
		t.Errorf("a query absent from the page should fall back to the lead, got %q", out)
	}
}

func TestPassages_ExtractReportsTruncation(t *testing.T) {
	ext := serveExtractor(t, htmlHandler(longGuide(), "text/html"))
	opts := &model.ExtractOptions{Mode: "markdown", MaxCharsPerPage: 1500, Query: "reciprocal rank fusion k constant"}
	content, err := ext.Extract(context.Background(), "http://example.com/guide", opts)
	if err != nil {
		t.Fatal(err)
	}
	if !content.Truncated || content.TotalChars <= 1500 || !strings.Contains(content.Markdown, "usually set to 60") {
		t.Fatalf("expected relevant, truncated content: truncated=%v total=%d\n%s", content.Truncated, content.TotalChars, content.Markdown)
	}
	if n := utf8.RuneCountInString(content.Markdown); n > 1500 {
		t.Fatalf("budget exceeded: %d chars", n)
	}

	// Without a query the lead is returned, still marked as truncated.
	opts.Query = ""
	lead, err := ext.Extract(context.Background(), "http://example.com/guide", opts)
	if err != nil {
		t.Fatal(err)
	}
	if !lead.Truncated || strings.Contains(lead.Markdown, "usually set to 60") || !strings.HasPrefix(lead.Markdown, "# Hybrid search guide") {
		t.Fatalf("expected the start of the page, got:\n%s", lead.Markdown)
	}

	// A page that fits is returned whole and not marked as truncated.
	opts.MaxCharsPerPage = 20000
	whole, err := ext.Extract(context.Background(), "http://example.com/guide", opts)
	if err != nil {
		t.Fatal(err)
	}
	if whole.Truncated || whole.TotalChars != utf8.RuneCountInString(whole.Markdown) || strings.Contains(whole.Markdown, GapMarker) {
		t.Fatalf("whole page expected: truncated=%v total=%d len=%d", whole.Truncated, whole.TotalChars, utf8.RuneCountInString(whole.Markdown))
	}
}

func TestPassages_LongParagraphsAreSplitAtSentences(t *testing.T) {
	long := strings.Repeat("Indexing pipelines tokenize documents before storing them. ", 40)
	chunks := buildChunks([]block{{md: long, text: long, splittable: true}}, chunkMaxChars)
	if len(chunks) < 2 {
		t.Fatalf("expected the paragraph to be split, got %d chunk(s)", len(chunks))
	}
	for _, c := range chunks {
		if n := utf8.RuneCountInString(c.md); n > chunkMaxChars {
			t.Errorf("chunk of %d chars exceeds %d", n, chunkMaxChars)
		}
		if !strings.HasSuffix(c.md, ".") {
			t.Errorf("chunk not cut at a sentence boundary: %q", c.md[len(c.md)-20:])
		}
	}
}

func TestPassages_LongListsAreSplitIntoItems(t *testing.T) {
	// Python's docs put each function in a <dl>; a whole entry must not become
	// one chunk too big to ever fit the budget.
	var b strings.Builder
	b.WriteString(`<html><body><article><h1>asyncio tasks</h1><dl class="py function"><dt>gather(*aws, return_exceptions=False)</dt><dd>`)
	for i := 0; i < 8; i++ {
		b.WriteString(`<p>Run awaitable objects concurrently and collect their results in order, waiting for all of them to finish first.</p>`)
	}
	b.WriteString(`<p>If return_exceptions is True, exceptions are treated the same as successful results and aggregated in the result list.</p></dd></dl><ol>`)
	for i := 0; i < 12; i++ {
		b.WriteString(`<li>Step of a long numbered procedure that keeps going for a while to make this list long.</li>`)
	}
	b.WriteString(`</ol></article></body></html>`)
	p, err := ParseHTML([]byte(b.String()))
	if err != nil {
		t.Fatal(err)
	}
	chunks := buildChunks(collectBlocks(p.MainNode, nil), chunkMaxChars)
	foundAnswer, foundNumber := false, false
	for _, c := range chunks {
		if n := utf8.RuneCountInString(c.md); n > chunkMaxChars+200 {
			t.Errorf("chunk of %d chars: long lists should be split", n)
		}
		foundAnswer = foundAnswer || strings.Contains(c.md, "If return_exceptions is True")
		foundNumber = foundNumber || strings.Contains(c.md, "12. Step of a long numbered procedure")
	}
	if !foundAnswer || !foundNumber {
		t.Fatalf("items lost or renumbered (answer=%v, numbering=%v)", foundAnswer, foundNumber)
	}
	out := passagesFor(t, b.String(), "gather return_exceptions", 700, true)
	if !strings.Contains(out, "If return_exceptions is True") {
		t.Fatalf("answer not selected:\n%s", out)
	}
}
