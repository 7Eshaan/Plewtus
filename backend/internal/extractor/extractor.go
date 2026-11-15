package extractor

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"strings"
	"sync"
	"time"
	"unicode/utf8"

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
		Title:         parsed.Metadata.Title,
		Author:        parsed.Metadata.Author,
		Description:   parsed.Metadata.Description,
		ExtractedAt:   time.Now().UTC().Format(time.RFC3339),
		PublishedDate: parsed.Metadata.PublishedDate,
	}

	mode := opts.Mode
	if mode == "" {
		mode = "highlights"
	}

	fullText := ToText(parsed.MainNode, 20000)
	if mode != "metadata" && utf8.RuneCountInString(fullText) < minReadableChars {
		// Next to nothing is left once page chrome is removed: typically an
		// app shell that renders its content with JavaScript (YouTube, SPAs).
		// The page's own description is the best summary available; without
		// one, report the page as unreadable instead of returning chrome.
		description := parsed.Metadata.Description
		if description == "" {
			return nil, ErrNoReadableContent
		}
		description = truncateCleanly(description, maxChars)
		switch mode {
		case "markdown":
			content.Markdown = description
		case "text":
			content.Text = description
		default:
			content.Highlights = []string{description}
		}
		return content, nil
	}

	// Each mode returns one representation of the page body; duplicating the
	// same content as both Markdown and text would double agent token usage.
	switch mode {
	case "markdown":
		base := parsed.ResolveBase(targetURL)
		content.Markdown = ToMarkdownWithBase(parsed.MainNode, maxChars, base)
		// The chosen container can start below the page heading (for example
		// the article body without its <h1>); lead with the page title then.
		if title := strings.TrimSpace(content.Title); title != "" && !startsWithHeading(content.Markdown) {
			heading := "# " + title + "\n\n"
			if budget := maxChars - utf8.RuneCountInString(heading); budget > maxChars/2 {
				content.Markdown = heading + ToMarkdownWithBase(parsed.MainNode, budget, base)
			}
		}

	case "text":
		content.Text = ToText(parsed.MainNode, maxChars)

	case "metadata":
		if parsed.Metadata.Description == "" {
			content.Text = truncateCleanly(fullText, 500)
		}

	case "highlights":
		fallthrough
	default:
		content.Highlights = ExtractHighlights(fullText, opts.Query, 4)
		if len(content.Highlights) == 0 {
			content.Text = truncateCleanly(fullText, maxChars)
		}
	}

	return content, nil
}

// ErrNoReadableContent reports a page with no extractable text, usually one
// that is rendered entirely with JavaScript.
var ErrNoReadableContent = errors.New("no readable text found on the page (it may need JavaScript to render)")

// minReadableChars is the least text a page must yield to count as readable.
const minReadableChars = 200

// startsWithHeading reports whether a heading appears near the top of md.
func startsWithHeading(md string) bool {
	head := md
	if len(head) > 300 {
		head = head[:300]
	}
	for _, line := range strings.Split(head, "\n") {
		if strings.HasPrefix(strings.TrimSpace(line), "#") {
			return true
		}
	}
	return false
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
				results[idx].ContentError = describeExtractError(ctx.Err())
				return
			}

			content, err := e.Extract(ctx, targetURL, opts)
			if err != nil {
				e.logger.Debug("page extraction skipped or failed",
					slog.String("url", targetURL),
					slog.Any("error", err),
				)
				results[idx].ContentError = describeExtractError(err)
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

// describeExtractError turns an extraction failure into a short reason that is
// safe to show API callers.
func describeExtractError(err error) string {
	switch {
	case errors.Is(err, context.DeadlineExceeded):
		return "timed out while fetching the page"
	case errors.Is(err, context.Canceled):
		return "extraction was cancelled"
	}
	msg := err.Error()
	// Drop the "fetch failed for <url>: " prefix; the result already has the URL.
	if _, rest, found := strings.Cut(msg, ": "); found && strings.HasPrefix(msg, "fetch failed for ") {
		msg = rest
	}
	if len(msg) > 200 {
		msg = msg[:200]
	}
	return msg
}
