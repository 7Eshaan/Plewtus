package extractor

import (
	"context"
	"testing"
	"time"

	"plewtus/internal/model"
)

func TestLiveWebpageExtraction(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping live webpage extraction test in short mode")
	}

	ext := New(ExtractorOptions{
		Timeout: 6 * time.Second,
	})

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Test 1: Markdown extraction on go.dev
	targetURL := "https://go.dev"
	contentMD, err := ext.Extract(ctx, targetURL, &model.ExtractOptions{
		Enabled:         true,
		Mode:            "markdown",
		MaxCharsPerPage: 2000,
	})
	if err != nil {
		t.Fatalf("live extraction failed for %s: %v", targetURL, err)
	}

	if contentMD == nil {
		t.Fatalf("expected non-nil content")
	}
	if contentMD.Markdown == "" {
		t.Errorf("expected non-empty markdown")
	}
	if contentMD.Text == "" {
		t.Errorf("expected non-empty text")
	}
	if len(contentMD.Markdown) > 2050 {
		t.Errorf("markdown exceeded max chars: %d", len(contentMD.Markdown))
	}
	t.Logf("Live Markdown from %s (length %d):\n%s\n", targetURL, len(contentMD.Markdown), contentMD.Markdown[:min(300, len(contentMD.Markdown))])

	// Test 2: Highlights extraction on go.dev/doc
	docURL := "https://go.dev/doc/"
	contentHL, err := ext.Extract(ctx, docURL, &model.ExtractOptions{
		Enabled:         true,
		Mode:            "highlights",
		MaxCharsPerPage: 2000,
	})
	if err != nil {
		t.Fatalf("live highlights extraction failed for %s: %v", docURL, err)
	}

	if contentHL == nil {
		t.Fatalf("expected non-nil content")
	}
	if len(contentHL.Highlights) == 0 {
		t.Errorf("expected at least one highlight sentence")
	}
	t.Logf("Live Highlights from %s (%d highlights): %v", docURL, len(contentHL.Highlights), contentHL.Highlights)

	// Test 3: Batch extraction
	results := []model.SearchResult{
		{ID: "1", Title: "Go Dev", URL: "https://go.dev"},
		{ID: "2", Title: "Go Doc", URL: "https://go.dev/doc/"},
		{ID: "3", Title: "Go Blog", URL: "https://go.dev/blog/"},
	}

	batchResult := ext.ExtractBatch(ctx, results, &model.ExtractOptions{
		Enabled:  true,
		Mode:     "markdown",
		MaxPages: 2, // only top 2 should be extracted
	})

	if batchResult[0].Content == nil {
		t.Errorf("expected content on batch result 0")
	}
	if batchResult[1].Content == nil {
		t.Errorf("expected content on batch result 1")
	}
	if batchResult[2].Content != nil {
		t.Errorf("result 2 exceeded MaxPages limit of 2, but content was extracted")
	}
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
