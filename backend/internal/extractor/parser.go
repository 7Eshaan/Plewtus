package extractor

import (
	"bytes"
	"net/url"
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
	// BaseHref is the document's <base href>, if any, used to resolve links.
	BaseHref string
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
	atom.Aside:    true,
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

// Chrome elements that are noise at page level but meaningful inside an
// article (e.g. <article><header><h1>Title</h1></header>).
var pageChromeTags = map[atom.Atom]bool{
	atom.Header: true,
}

// noiseTokens are matched against the hyphen/underscore-separated parts of
// class and id names (so "cookie-banner" matches but "has-sidebar" layout
// wrappers and words like "download" do not trigger substring matches).
var noiseTokens = map[string]bool{
	"cookie": true, "cookies": true, "consent": true, "gdpr": true,
	"banner": true, "popup": true, "modal": true, "overlay": true,
	"sidebar": true, "advert": true, "advertisement": true, "ad": true, "ads": true,
	"disclaimer": true, "promo": true, "sponsor": true, "sponsored": true,
	"newsletter": true, "share": true, "social": true, "widget": true,
	"breadcrumb": true, "breadcrumbs": true, "related": true, "comments": true,
}

// layoutPrefixes mark state/layout classes such as "has-sidebar" or
// "no-cookie-banner" that describe a wrapper rather than being noise.
var layoutPrefixes = map[string]bool{
	"has": true, "with": true, "no": true, "without": true, "is": true, "show": true, "hide": true,
}

// ParseHTML parses raw HTML bytes, extracts metadata, prunes noise elements,
// and identifies the primary content area.
func ParseHTML(htmlBytes []byte) (*ParsedPage, error) {
	doc, err := html.Parse(bytes.NewReader(htmlBytes))
	if err != nil {
		return nil, err
	}

	meta := extractMetadata(doc)
	baseHref := findBaseHref(doc)

	// Clean document tree by pruning disallowed and noise nodes
	cleanNode(doc, false)

	// Locate the best candidate main content container
	mainNode := findMainContent(doc)

	return &ParsedPage{
		Metadata: meta,
		MainNode: mainNode,
		BaseHref: baseHref,
	}, nil
}

// ResolveBase returns the URL relative links should be resolved against: the
// document's <base href> (itself resolved against the page URL) or the page URL.
func (p *ParsedPage) ResolveBase(pageURL string) *url.URL {
	page, err := url.Parse(pageURL)
	if err != nil || page.Host == "" {
		return nil
	}
	if p.BaseHref != "" {
		if base, err := page.Parse(p.BaseHref); err == nil {
			return base
		}
	}
	return page
}

func findBaseHref(doc *html.Node) string {
	var href string
	var walk func(*html.Node)
	walk = func(n *html.Node) {
		if href != "" {
			return
		}
		if n.Type == html.ElementNode && n.DataAtom == atom.Base {
			href = strings.TrimSpace(getAttr(n, "href"))
			return
		}
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			walk(c)
		}
	}
	walk(doc)
	return href
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
// inContent is true below <main>/<article>, where headers carry the title.
func cleanNode(n *html.Node, inContent bool) {
	var next *html.Node
	for c := n.FirstChild; c != nil; c = next {
		next = c.NextSibling

		if shouldPruneNode(c, inContent) {
			n.RemoveChild(c)
			continue
		}
		childInContent := inContent || (c.Type == html.ElementNode && (c.DataAtom == atom.Main || c.DataAtom == atom.Article))
		cleanNode(c, childInContent)
	}
}

// shouldPruneNode determines if a node is noise (scripts, headers, nav, ads).
func shouldPruneNode(n *html.Node, inContent bool) bool {
	if n.Type != html.ElementNode {
		return false
	}

	if disallowedTags[n.DataAtom] {
		return true
	}
	if pageChromeTags[n.DataAtom] && !inContent {
		return true
	}

	// Never prune main content elements regardless of class
	if n.DataAtom == atom.Main || n.DataAtom == atom.Article || n.DataAtom == atom.Body || n.DataAtom == atom.Html {
		return false
	}

	if !hasNoiseName(getAttr(n, "class")) && !hasNoiseName(getAttr(n, "id")) {
		return false
	}
	// A "noisy" wrapper that encloses the primary content is a layout
	// container, not noise; keep it so the content survives.
	return !containsPrimaryContent(n)
}

func hasNoiseName(value string) bool {
	for _, name := range strings.Fields(strings.ToLower(value)) {
		parts := strings.FieldsFunc(name, func(r rune) bool { return r == '-' || r == '_' })
		if len(parts) == 0 || layoutPrefixes[parts[0]] {
			continue
		}
		for _, part := range parts {
			if noiseTokens[part] {
				return true
			}
		}
	}
	return false
}

func containsPrimaryContent(n *html.Node) bool {
	for c := n.FirstChild; c != nil; c = c.NextSibling {
		if c.Type == html.ElementNode {
			if c.DataAtom == atom.Main || c.DataAtom == atom.Article || strings.EqualFold(getAttr(c, "role"), "main") {
				return true
			}
			if containsPrimaryContent(c) {
				return true
			}
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
