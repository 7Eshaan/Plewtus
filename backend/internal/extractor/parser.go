package extractor

import (
	"bytes"
	"strings"

	"golang.org/x/net/html"
	"golang.org/x/net/html/atom"
)

// PageMetadata holds extracted semantic metadata from a webpage.
type PageMetadata struct {
	Title         string
	Description   string
	Author        string
	PublishedDate string
}

// ParsedPage represents the processed webpage ready for text/markdown conversion.
type ParsedPage struct {
	Metadata PageMetadata
	MainNode *html.Node
}

// Disallowed tags that provide no semantic value to AI agents
var disallowedTags = map[atom.Atom]bool{
	atom.Script:   true,
	atom.Style:    true,
	atom.Noscript: true,
	atom.Iframe:   true,
	atom.Svg:      true,
	atom.Canvas:   true,
	atom.Nav:      true,
	atom.Footer:   true,
	atom.Header:   true,
	atom.Aside:    true,
	atom.Form:     true,
	atom.Button:   true,
	atom.Input:    true,
	atom.Select:   true,
	atom.Textarea: true,
	atom.Video:    true,
	atom.Audio:    true,
	atom.Object:   true,
	atom.Embed:    true,
	atom.Applet:   true,
	atom.Menu:     true,
	atom.Dialog:   true,
}

// Noise patterns commonly found in boilerplates, ads, popups, and sidebars
var noisePatterns = []string{
	"cookie", "banner", "popup", "modal", "sidebar", "advert",
	"disclaimer", "promo", "sponsor", "newsletter", "share-button",
	"social-share", "widget", "comment-form",
}

// ParseHTML parses raw HTML bytes, extracts metadata, prunes noise elements,
// and identifies the primary content area.
func ParseHTML(htmlBytes []byte) (*ParsedPage, error) {
	doc, err := html.Parse(bytes.NewReader(htmlBytes))
	if err != nil {
		return nil, err
	}

	meta := extractMetadata(doc)

	// Clean document tree by pruning disallowed and noise nodes
	cleanNode(doc)

	// Locate the best candidate main content container
	mainNode := findMainContent(doc)

	return &ParsedPage{
		Metadata: meta,
		MainNode: mainNode,
	}, nil
}

// extractMetadata collects title, description, author, and publication date from head tags.
func extractMetadata(doc *html.Node) PageMetadata {
	var meta PageMetadata
	var titleTagText string

	var walk func(*html.Node)
	walk = func(n *html.Node) {
		if n.Type == html.ElementNode {
			switch n.DataAtom {
			case atom.Title:
				if titleTagText == "" {
					titleTagText = extractNodeText(n)
				}
			case atom.Meta:
				name := strings.ToLower(getAttr(n, "name"))
				property := strings.ToLower(getAttr(n, "property"))
				content := strings.TrimSpace(getAttr(n, "content"))

				if content != "" {
					// Title
					if meta.Title == "" {
						if property == "og:title" || name == "twitter:title" {
							meta.Title = content
						}
					}

					// Description
					if meta.Description == "" {
						if name == "description" || property == "og:description" || name == "twitter:description" {
							meta.Description = content
						}
					}

					// Author
					if meta.Author == "" {
						if name == "author" || property == "article:author" || name == "article:author" {
							meta.Author = content
						}
					}

					// Published date
					if meta.PublishedDate == "" {
						if property == "article:published_time" || name == "date" || name == "pubdate" || property == "og:updated_time" {
							meta.PublishedDate = content
						}
					}
				}
			case atom.Time:
				if meta.PublishedDate == "" {
					dt := getAttr(n, "datetime")
					if dt != "" {
						meta.PublishedDate = strings.TrimSpace(dt)
					}
				}
			}
		}

		for c := n.FirstChild; c != nil; c = c.NextSibling {
			walk(c)
		}
	}

	walk(doc)

	if meta.Title == "" && titleTagText != "" {
		meta.Title = titleTagText
	}

	return meta
}

// cleanNode removes noise elements (scripts, styles, nav, ads) in-place from the DOM.
func cleanNode(n *html.Node) {
	var next *html.Node
	for c := n.FirstChild; c != nil; c = next {
		next = c.NextSibling

		if shouldPruneNode(c) {
			n.RemoveChild(c)
		} else {
			cleanNode(c)
		}
	}
}

// shouldPruneNode determines if a node is noise (scripts, headers, nav, ads).
func shouldPruneNode(n *html.Node) bool {
	if n.Type != html.ElementNode {
		return false
	}

	if disallowedTags[n.DataAtom] {
		return true
	}

	// Never prune main content elements regardless of class
	if n.DataAtom == atom.Main || n.DataAtom == atom.Article || n.DataAtom == atom.Body {
		return false
	}

	classVal := strings.ToLower(getAttr(n, "class"))
	idVal := strings.ToLower(getAttr(n, "id"))
	combined := classVal + " " + idVal

	for _, pattern := range noisePatterns {
		if strings.Contains(combined, pattern) {
			return true
		}
	}

	return false
}

// findMainContent locates the primary content node (<main>, <article>, or <body> fallback).
func findMainContent(doc *html.Node) *html.Node {
	var mainCandidates []*html.Node
	var bodyNode *html.Node

	var walk func(*html.Node)
	walk = func(n *html.Node) {
		if n.Type == html.ElementNode {
			if n.DataAtom == atom.Body {
				bodyNode = n
			}

			// Prioritize semantic tags
			if n.DataAtom == atom.Main || n.DataAtom == atom.Article {
				mainCandidates = append(mainCandidates, n)
			} else {
				role := strings.ToLower(getAttr(n, "role"))
				id := strings.ToLower(getAttr(n, "id"))
				if role == "main" || id == "main" || id == "content" || id == "main-content" {
					mainCandidates = append(mainCandidates, n)
				}
			}
		}

		for c := n.FirstChild; c != nil; c = c.NextSibling {
			walk(c)
		}
	}

	walk(doc)

	if len(mainCandidates) > 0 {
		// Pick candidate with greatest text density/length
		best := mainCandidates[0]
		bestLen := len(extractNodeText(best))
		for _, cand := range mainCandidates[1:] {
			candLen := len(extractNodeText(cand))
			if candLen > bestLen {
				best = cand
				bestLen = candLen
			}
		}
		if bestLen > 100 {
			return best
		}
	}

	if bodyNode != nil {
		return bodyNode
	}

	return doc
}

// extractNodeText recursively extracts raw text from a node.
func extractNodeText(n *html.Node) string {
	var buf strings.Builder
	var walk func(*html.Node)
	walk = func(node *html.Node) {
		if node.Type == html.TextNode {
			buf.WriteString(node.Data)
			buf.WriteString(" ")
		}
		for c := node.FirstChild; c != nil; c = c.NextSibling {
			walk(c)
		}
	}
	walk(n)
	return strings.Join(strings.Fields(buf.String()), " ")
}

// getAttr returns the value of an attribute on a node by key.
func getAttr(n *html.Node, key string) string {
	for _, a := range n.Attr {
		if strings.EqualFold(a.Key, key) {
			return a.Val
		}
	}
	return ""
}
