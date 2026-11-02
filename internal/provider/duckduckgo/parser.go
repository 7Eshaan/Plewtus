package duckduckgo

import (
	"bytes"
	"errors"
	"io"
	"strings"

	"golang.org/x/net/html"

	"plewtus/internal/model"
	"plewtus/internal/provider/htmlutil"
)

var (
	ErrChallengeEncountered = errors.New("duckduckgo challenge/captcha encountered")
	ErrNoResultsFound       = errors.New("no results found on duckduckgo")
)

// ParseHTMLResponse parses DuckDuckGo /html/ endpoint response.
func ParseHTMLResponse(r io.Reader, limit int) ([]model.RawSearchResult, error) {
	bodyBytes, err := io.ReadAll(r)
	if err != nil {
		return nil, err
	}

	bodyStr := string(bodyBytes)
	if isChallengeResponse(bodyStr) {
		return nil, ErrChallengeEncountered
	}

	doc, err := html.Parse(bytes.NewReader(bodyBytes))
	if err != nil {
		return nil, err
	}

	// In /html/, each organic result is in div.result.web-result or div.result
	resultNodes := htmlutil.FindAll(doc, func(n *html.Node) bool {
		if n.Type != html.ElementNode || n.Data != "div" {
			return false
		}
		if !htmlutil.HasClass(n, "result") {
			return false
		}
		// Skip ads
		if htmlutil.HasAnyClass(n, "result--ad", "result--sponsored", "badge--ad") {
			return false
		}
		return true
	})

	var results []model.RawSearchResult
	position := 1

	for _, node := range resultNodes {
		if limit > 0 && len(results) >= limit {
			break
		}

		// Find title and link inside .result__a
		titleNode := htmlutil.FindOne(node, func(n *html.Node) bool {
			return n.Type == html.ElementNode && n.Data == "a" && htmlutil.HasClass(n, "result__a")
		})
		if titleNode == nil {
			continue
		}

		href, ok := htmlutil.GetAttr(titleNode, "href")
		if !ok || href == "" {
			continue
		}

		// Skip ad/sponsored redirect URLs
		if strings.Contains(href, "duckduckgo.com/y.js") || strings.Contains(href, "ad_provider=") {
			continue
		}

		rawURL := htmlutil.UnwrapURL(href)
		if rawURL == "" || !strings.HasPrefix(rawURL, "http") {
			continue
		}

		title := strings.TrimSpace(htmlutil.TextContent(titleNode))
		if title == "" {
			continue
		}

		// Find snippet in .result__snippet
		var snippet string
		snippetNode := htmlutil.FindOne(node, func(n *html.Node) bool {
			return n.Type == html.ElementNode && htmlutil.HasClass(n, "result__snippet")
		})
		if snippetNode != nil {
			snippet = strings.TrimSpace(htmlutil.TextContent(snippetNode))
		}

		results = append(results, model.RawSearchResult{
			Title:    title,
			URL:      rawURL,
			Snippet:  snippet,
			Position: position,
		})
		position++
	}

	return results, nil
}

// ParseLiteResponse parses DuckDuckGo /lite/ endpoint response.
func ParseLiteResponse(r io.Reader, limit int) ([]model.RawSearchResult, error) {
	bodyBytes, err := io.ReadAll(r)
	if err != nil {
		return nil, err
	}

	bodyStr := string(bodyBytes)
	if isChallengeResponse(bodyStr) {
		return nil, ErrChallengeEncountered
	}

	doc, err := html.Parse(bytes.NewReader(bodyBytes))
	if err != nil {
		return nil, err
	}

	// Find all <a> with class 'result-link'
	linkNodes := htmlutil.FindAll(doc, func(n *html.Node) bool {
		return n.Type == html.ElementNode && n.Data == "a" && htmlutil.HasClass(n, "result-link")
	})

	var results []model.RawSearchResult
	position := 1

	for _, linkNode := range linkNodes {
		if limit > 0 && len(results) >= limit {
			break
		}

		href, ok := htmlutil.GetAttr(linkNode, "href")
		if !ok || href == "" {
			continue
		}

		// Skip sponsored links
		if strings.Contains(href, "duckduckgo.com/y.js") || strings.Contains(href, "ad_provider=") {
			continue
		}

		title := strings.TrimSpace(htmlutil.TextContent(linkNode))
		if title == "" || strings.EqualFold(title, "more info") {
			continue
		}

		rawURL := htmlutil.UnwrapURL(href)
		if rawURL == "" || !strings.HasPrefix(rawURL, "http") {
			continue
		}

		// Look for snippet: in lite, snippet is in the succeeding <tr> with class 'result-snippet'
		snippet := findLiteSnippet(linkNode)

		results = append(results, model.RawSearchResult{
			Title:    title,
			URL:      rawURL,
			Snippet:  snippet,
			Position: position,
		})
		position++
	}

	return results, nil
}

// findLiteSnippet traverses the table rows after a result link to locate the snippet cell.
func findLiteSnippet(linkNode *html.Node) string {
	// Find enclosing <tr>
	tr := linkNode
	for tr != nil && !(tr.Type == html.ElementNode && tr.Data == "tr") {
		tr = tr.Parent
	}
	if tr == nil {
		return ""
	}

	// Traverse next siblings of <tr>
	for nextTR := tr.NextSibling; nextTR != nil; nextTR = nextTR.NextSibling {
		if nextTR.Type != html.ElementNode || nextTR.Data != "tr" {
			continue
		}

		// Check if this row contains another result-link (i.e. start of next result)
		hasResultLink := htmlutil.FindOne(nextTR, func(n *html.Node) bool {
			return n.Type == html.ElementNode && n.Data == "a" && htmlutil.HasClass(n, "result-link")
		})
		if hasResultLink != nil {
			break
		}

		// Look for .result-snippet td
		snippetTD := htmlutil.FindOne(nextTR, func(n *html.Node) bool {
			return n.Type == html.ElementNode && n.Data == "td" && htmlutil.HasClass(n, "result-snippet")
		})
		if snippetTD != nil {
			return strings.TrimSpace(htmlutil.TextContent(snippetTD))
		}
	}

	return ""
}

func isChallengeResponse(htmlContent string) bool {
	return strings.Contains(htmlContent, "anomaly-modal") ||
		strings.Contains(htmlContent, "anomaly.js") ||
		strings.Contains(htmlContent, "Select all squares containing a duck") ||
		strings.Contains(htmlContent, "Unfortunately, bots use DuckDuckGo too")
}
