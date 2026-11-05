package extractor

import (
	"context"
	"fmt"
	"log/slog"
	"sync"
	"time"

	"plewtus/internal/model"
)

// ContentExtractor defines the interface for webpage content extraction.
type ContentExtractor interface {
	Extract(ctx context.Context, targetURL string, opts *model.ExtractOptions) (*model.ExtractedContent, error)
	ExtractBatch(ctx context.Context, results []model.SearchResult, opts *model.ExtractOptions) []model.SearchResult
}

// Extractor handles safe webpage fetching and AI-friendly content conversion.
type Extractor struct {
	fetcher       *Fetcher
	logger        *slog.Logger
	timeout       time.Duration
	maxConcurrent int
}

// ExtractorOptions configures the Extractor.
type ExtractorOptions struct {
	Timeout       time.Duration
	MaxBodyBytes  int64
	UserAgent     string
	MaxConcurrent int
	Fetcher       *Fetcher
	Logger        *slog.Logger
}

// New creates a new Extractor instance.
func New(opts ExtractorOptions) *Extractor {
	timeout := opts.Timeout
	if timeout <= 0 {
		timeout = DefaultFetchTimeout
	}

	logger := opts.Logger
	if logger == nil {
		logger = slog.Default()
	}

	maxConcurrent := opts.MaxConcurrent
	if maxConcurrent <= 0 {
		maxConcurrent = 5
	}

	fetcher := opts.Fetcher
	if fetcher == nil {
		fetcher = NewFetcher(FetcherOptions{
			Timeout:      timeout,
			MaxBodyBytes: opts.MaxBodyBytes,
			UserAgent:    opts.UserAgent,
		})
	}

	return &Extractor{
		fetcher:       fetcher,
		logger:        logger,
		timeout:       timeout,
		maxConcurrent: maxConcurrent,
	}
}

// Extract retrieves and formats content from a single webpage according to options.
func (e *Extractor) Extract(ctx context.Context, targetURL string, opts *model.ExtractOptions) (*model.ExtractedContent, error) {
	if opts == nil {
		opts = &model.ExtractOptions{
			Enabled:         true,
			Mode:            "highlights",
			MaxCharsPerPage: 4000,
		}
	}

	// Bounded context for this extraction
	extractCtx, cancel := context.WithTimeout(ctx, e.timeout)
	defer cancel()

	htmlBytes, _, err := e.fetcher.Fetch(extractCtx, targetURL)
	if err != nil {
		return nil, fmt.Errorf("fetch failed for %s: %w", targetURL, err)
	}

	parsed, err := ParseHTML(htmlBytes)
	if err != nil {
		return nil, fmt.Errorf("html parse failed for %s: %w", targetURL, err)
	}

	maxChars := opts.MaxCharsPerPage
	if maxChars <= 0 {
		maxChars = 4000
	}

	content := &model.ExtractedContent{
		Author:        parsed.Metadata.Author,
		Description:   parsed.Metadata.Description,
		ExtractedAt:   time.Now().UTC().Format(time.RFC3339),
		PublishedDate: parsed.Metadata.PublishedDate,
	}

	mode := opts.Mode
	if mode == "" {
		mode = "highlights"
	}

	switch mode {
	case "markdown":
		content.Markdown = ToMarkdown(parsed.MainNode, maxChars)
		content.Text = ToText(parsed.MainNode, maxChars)

	case "text":
		content.Text = ToText(parsed.MainNode, maxChars)

	case "metadata":
		if parsed.Metadata.Description != "" {
			content.Text = parsed.Metadata.Description
		} else {
			content.Text = ToText(parsed.MainNode, 500)
		}

	case "highlights":
		fallthrough
	default:
		fullText := ToText(parsed.MainNode, 10000)
		highlights := ExtractHighlights(fullText, opts.Query, 4)
		content.Highlights = highlights
		if len(highlights) > 0 {
			content.Text = truncateCleanly(fullText, maxChars)
		} else {
			content.Text = ToText(parsed.MainNode, maxChars)
		}
	}

	return content, nil
}

// ExtractBatch extracts content concurrently for the top results up to opts.MaxPages.
func (e *Extractor) ExtractBatch(ctx context.Context, results []model.SearchResult, opts *model.ExtractOptions) []model.SearchResult {
	if opts == nil || !opts.Enabled || len(results) == 0 {
		return results
	}

	maxPages := opts.MaxPages
	if maxPages <= 0 {
		maxPages = 3
	}
	if maxPages > len(results) {
		maxPages = len(results)
	}

	// We only process pages up to maxPages to avoid unnecessary network queries
	sem := make(chan struct{}, e.maxConcurrent)
	var wg sync.WaitGroup

	for i := 0; i < maxPages; i++ {
		idx := i
		targetURL := results[idx].URL

		wg.Add(1)
		go func() {
			defer wg.Done()

			select {
			case sem <- struct{}{}:
				defer func() { <-sem }()
			case <-ctx.Done():
				return
			}

			content, err := e.Extract(ctx, targetURL, opts)
			if err != nil {
				e.logger.Debug("page extraction skipped or failed",
					slog.String("url", targetURL),
					slog.Any("error", err),
				)
				return
			}

			results[idx].Content = content
			if results[idx].PublishedDate == "" {
				results[idx].PublishedDate = content.PublishedDate
			}
		}()
	}

	wg.Wait()
	return results
}
