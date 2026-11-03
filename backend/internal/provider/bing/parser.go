package bing

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
	ErrChallengeEncountered = errors.New("bing challenge/captcha encountered")
	ErrBlocked              = errors.New("bing request blocked")
)

// ParseHTMLResponse parses Bing SERP HTML response.
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

	// In Bing, organic results are enclosed in <li class="b_algo">
	algoNodes := htmlutil.FindAll(doc, func(n *html.Node) bool {
		if n.Type != html.ElementNode || n.Data != "li" {
			return false
		}
		return htmlutil.HasClass(n, "b_algo")
	})

	var results []model.RawSearchResult
	position := 1

	for _, node := range algoNodes {
		if limit > 0 && len(results) >= limit {
			break
		}

		// Title and URL in <h2><a href="...">
		h2Node := htmlutil.FindOne(node, func(n *html.Node) bool {
			return n.Type == html.ElementNode && n.Data == "h2"
		})
		if h2Node == nil {
			continue
		}

		aNode := htmlutil.FindOne(h2Node, func(n *html.Node) bool {
			return n.Type == html.ElementNode && n.Data == "a"
		})
		if aNode == nil {
			continue
		}

		rawTitle := strings.TrimSpace(htmlutil.TextContent(aNode))
		if rawTitle == "" {
			continue
		}

		href, _ := htmlutil.GetAttr(aNode, "href")
		targetURL := htmlutil.UnwrapURL(href)
		if targetURL == "" || !strings.HasPrefix(targetURL, "http") {
			// Fallback: check .b_attribution cite
			citeNode := htmlutil.FindOne(node, func(n *html.Node) bool {
				return n.Type == html.ElementNode && n.Data == "cite"
			})
			if citeNode != nil {
				citeText := strings.TrimSpace(htmlutil.TextContent(citeNode))
				// cite might be like "https://go.dev › wiki"
				citeParts := strings.Fields(citeText)
				if len(citeParts) > 0 && strings.HasPrefix(citeParts[0], "http") {
					targetURL = citeParts[0]
				}
			}
		}

		if targetURL == "" || !strings.HasPrefix(targetURL, "http") {
			continue
		}

		// Snippet in .b_caption or .b_caption p
		var snippet string
		var publishedDate string

		captionNode := htmlutil.FindOne(node, func(n *html.Node) bool {
			return n.Type == html.ElementNode && htmlutil.HasClass(n, "b_caption")
		})
		if captionNode != nil {
			// Check for date in .news_dt
			dateNode := htmlutil.FindOne(captionNode, func(n *html.Node) bool {
				return n.Type == html.ElementNode && htmlutil.HasClass(n, "news_dt")
			})
			if dateNode != nil {
				publishedDate = strings.TrimSpace(htmlutil.TextContent(dateNode))
			}

			// Snippet paragraph
			pNode := htmlutil.FindOne(captionNode, func(n *html.Node) bool {
				return n.Type == html.ElementNode && (n.Data == "p" || htmlutil.HasClass(n, "b_lineclamp2"))
			})
			if pNode != nil {
				snippet = strings.TrimSpace(htmlutil.TextContent(pNode))
				// If snippet has date prefix, remove or keep it
				if publishedDate != "" && strings.HasPrefix(snippet, publishedDate) {
					snippet = strings.TrimSpace(strings.TrimPrefix(snippet, publishedDate))
					snippet = strings.TrimPrefix(snippet, "·")
					snippet = strings.TrimSpace(snippet)
				}
			} else {
				snippet = strings.TrimSpace(htmlutil.TextContent(captionNode))
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
	return strings.Contains(lower, "please verify you are a human") ||
		strings.Contains(lower, "unusual traffic from your computer network") ||
		strings.Contains(lower, "our systems have detected unusual traffic")
}
