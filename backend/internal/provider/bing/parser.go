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

var ErrChallengeEncountered = errors.New("bing challenge/captcha encountered")

// ParseHTMLResponse parses Bing SERP HTML response.
func ParseHTMLResponse(r io.Reader, limit int) ([]model.RawSearchResult, error) {
	bodyBytes, err := io.ReadAll(r)
	if err != nil {
		return nil, err
	}

	doc, err := html.Parse(bytes.NewReader(bodyBytes))
	if err != nil {
		return nil, err
	}

	// In Bing, organic results are enclosed in <li class="b_algo">
	algoNodes := htmlutil.FindAllInAncestorScope(doc, "b_results", func(n *html.Node) bool {
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

	// A block page has no organic results. Checking only then avoids treating a
	// real result that merely quotes the block-page wording as a challenge.
	if len(results) == 0 && isBlockedOrChallenge(bodyBytes) {
		return nil, ErrChallengeEncountered
	}
	return results, nil
}

func isBlockedOrChallenge(body []byte) bool {
	lower := bytes.ToLower(body)
	return bytes.Contains(lower, []byte("please verify you are a human")) ||
		bytes.Contains(lower, []byte("unusual traffic from your computer network")) ||
		bytes.Contains(lower, []byte("our systems have detected unusual traffic"))
}
