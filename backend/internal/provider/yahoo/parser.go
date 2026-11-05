package yahoo

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
	ErrChallengeEncountered = errors.New("yahoo challenge/captcha encountered")
)

// ParseHTMLResponse parses Yahoo SERP HTML response.
func ParseHTMLResponse(r io.Reader, limit int) ([]model.RawSearchResult, error) {
	bodyBytes, err := io.ReadAll(r)
	if err != nil {
		return nil, err
	}

	bodyStr := string(bodyBytes)
	if isBlockedOrChallenge(bodyStr) {
		return nil, ErrChallengeEncountered
	}

	doc, err := html.Parse(bytes.NewReader(bodyBytes))
	if err != nil {
		return nil, err
	}

	// In Yahoo, organic results have class "algo" or "algo-sr"
	algoNodes := htmlutil.FindAll(doc, func(n *html.Node) bool {
		if n.Type != html.ElementNode || n.Data != "div" {
			return false
		}
		if !htmlutil.HasClass(n, "algo") {
			return false
		}
		if !htmlutil.HasAncestorID(n, "web") {
			return false
		}
		// Skip ads
		if htmlutil.HasAnyClass(n, "ad", "ads", "algo-ad") {
			return false
		}
		return true
	})

	var results []model.RawSearchResult
	position := 1

	for _, node := range algoNodes {
		if limit > 0 && len(results) >= limit {
			break
		}

		// Find anchor tag for the title and link
		var aNode *html.Node

		// First try inside .compTitle a
		compTitleNode := htmlutil.FindOne(node, func(n *html.Node) bool {
			return n.Type == html.ElementNode && htmlutil.HasClass(n, "compTitle")
		})
		if compTitleNode != nil {
			aNode = htmlutil.FindOne(compTitleNode, func(n *html.Node) bool {
				return n.Type == html.ElementNode && n.Data == "a"
			})
		}

		// Fallback: search for h3.title a or h3 a
		if aNode == nil {
			h3Node := htmlutil.FindOne(node, func(n *html.Node) bool {
				return n.Type == html.ElementNode && n.Data == "h3"
			})
			if h3Node != nil {
				aNode = htmlutil.FindOne(h3Node, func(n *html.Node) bool {
					return n.Type == html.ElementNode && n.Data == "a"
				})
			}
		}

		if aNode == nil {
			continue
		}

		href, _ := htmlutil.GetAttr(aNode, "href")
		targetURL := htmlutil.UnwrapURL(href)
		if targetURL == "" || !strings.HasPrefix(targetURL, "http") {
			continue
		}

		// Title extraction: check h3 inside a or a text
		var rawTitle string
		h3Node := htmlutil.FindOne(aNode, func(n *html.Node) bool {
			return n.Type == html.ElementNode && n.Data == "h3"
		})
		if h3Node != nil {
			rawTitle = strings.TrimSpace(htmlutil.TextContent(h3Node))
		}
		if rawTitle == "" {
			rawTitle = strings.TrimSpace(htmlutil.TextContent(aNode))
		}
		if rawTitle == "" {
			continue
		}

		// Snippet in .compText or .compText p
		var snippet string
		var publishedDate string

		compTextNode := htmlutil.FindOne(node, func(n *html.Node) bool {
			return n.Type == html.ElementNode && htmlutil.HasClass(n, "compText")
		})
		if compTextNode != nil {
			// Check for date in .fc-smoke
			dateNode := htmlutil.FindOne(compTextNode, func(n *html.Node) bool {
				return n.Type == html.ElementNode && htmlutil.HasClass(n, "fc-smoke")
			})
			if dateNode != nil {
				publishedDate = strings.TrimSpace(htmlutil.TextContent(dateNode))
				publishedDate = strings.TrimRight(publishedDate, "· -")
				publishedDate = strings.TrimSpace(publishedDate)
			}

			snippet = strings.TrimSpace(htmlutil.TextContent(compTextNode))
			if publishedDate != "" && strings.HasPrefix(snippet, publishedDate) {
				snippet = strings.TrimSpace(strings.TrimPrefix(snippet, publishedDate))
				snippet = strings.TrimPrefix(snippet, "·")
				snippet = strings.TrimPrefix(snippet, "-")
				snippet = strings.TrimSpace(snippet)
			}
		}

		results = append(results, model.RawSearchResult{
			Title:         rawTitle,
			URL:           targetURL,
			Snippet:       snippet,
			Position:      position,
			PublishedDate: publishedDate,
		})
		position++
	}

	return results, nil
}

func isBlockedOrChallenge(htmlContent string) bool {
	lower := strings.ToLower(htmlContent)
	return strings.Contains(lower, "robot check") ||
		strings.Contains(lower, "will be unblocked shortly") ||
		strings.Contains(lower, "verify you are human")
}
