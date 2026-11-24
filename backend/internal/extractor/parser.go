package extractor

import (
	"bytes"
	"net/url"
	"strings"
	"unicode"

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
	// <template> holds inert markup for client-side rendering (often with
	// ${placeholders}); it is never shown as page content.
	atom.Template: true,
}

// noiseRoles are ARIA landmark and widget roles that mark page chrome.
var noiseRoles = map[string]bool{
	"navigation": true, "contentinfo": true, "complementary": true, "search": true,
	"menu": true, "menubar": true, "dialog": true, "alertdialog": true, "toolbar": true,
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
	// Navigation chrome: site menus, language pickers ("vector-menu"),
	// dropdowns, skip links, and sign-in prompts.
	"nav": true, "navbar": true, "navigation": true, "menu": true, "dropdown": true,
	"toolbar": true, "skip": true, "login": true, "signin": true,
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

	// Drop blocks that are mostly links (menus, language lists, link farms)
	// before choosing the content container, so chrome does not win.
	pruneDocument(doc, true)
	mainNode := findMainContent(doc)

	// On pages that are themselves link collections (reading lists, indexes)
	// removing link lists leaves next to nothing; keep the links there.
	if countText(mainNode) < minReadableChars {
		withLinks, err := html.Parse(bytes.NewReader(htmlBytes))
		if err != nil {
			return nil, err
		}
		pruneDocument(withLinks, false)
		if candidate := findMainContent(withLinks); countText(candidate) > countText(mainNode) {
			mainNode = candidate
		}
	}

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
func cleanNode(n *html.Node, inContent bool, holdsMostText func(*html.Node) bool) {
	var next *html.Node
	for c := n.FirstChild; c != nil; c = next {
		next = c.NextSibling

		if shouldPruneNode(c, inContent) && (disallowedTags[c.DataAtom] || !holdsMostText(c)) {
			n.RemoveChild(c)
			continue
		}
		childInContent := inContent || (c.Type == html.ElementNode && (c.DataAtom == atom.Main || c.DataAtom == atom.Article))
		cleanNode(c, childInContent, holdsMostText)
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

	// The `hidden` attribute is deliberately not treated as noise: streaming
	// renderers (Next.js) deliver the article inside <div hidden> and reveal
	// it with JavaScript.
	hidden := strings.EqualFold(strings.TrimSpace(getAttr(n, "aria-hidden")), "true")
	noiseRole := noiseRoles[strings.ToLower(strings.TrimSpace(getAttr(n, "role")))]
	if !hidden && !noiseRole && !hasNoiseName(getAttr(n, "class")) && !hasNoiseName(getAttr(n, "id")) {
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

// textStats counts the visible characters below a node (whitespace
// excluded) and how many of them sit inside links.
type textStats struct {
	total, link int
	links       int // number of <a> elements with text
}

func (s textStats) prose() int { return s.total - s.link }

func (s textStats) linkDensity() float64 {
	if s.total == 0 {
		return 0
	}
	return float64(s.link) / float64(s.total)
}

// collectTextStats computes textStats for every element in one bottom-up pass.
func collectTextStats(root *html.Node) map[*html.Node]textStats {
	stats := make(map[*html.Node]textStats)
	var walk func(n *html.Node, inLink bool) textStats
	walk = func(n *html.Node, inLink bool) textStats {
		if n.Type == html.TextNode {
			count := 0
			for _, r := range n.Data {
				if !unicode.IsSpace(r) {
					count++
				}
			}
			s := textStats{total: count}
			if inLink {
				s.link = count
			}
			return s
		}
		if n.Type == html.ElementNode && disallowedTags[n.DataAtom] {
			return textStats{} // script/style source is not page text
		}
		childInLink := inLink || (n.Type == html.ElementNode && n.DataAtom == atom.A)
		var s textStats
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			cs := walk(c, childInLink)
			s.total += cs.total
			s.link += cs.link
			s.links += cs.links
		}
		if n.Type == html.ElementNode && n.DataAtom == atom.A && !inLink && s.total > 0 {
			s.links++
		}
		if n.Type == html.ElementNode {
			stats[n] = s
		}
		return s
	}
	stats[root] = walk(root, false)
	return stats
}

// Link-heavy blocks are removed when they hold several links, at least this
// share of their text is link text, and they carry little prose of their own.
// A single linked card or heading is not a menu.
const (
	linkHeavyDensity  = 0.5
	linkHeavyMaxProse = 300
	linkHeavyMinLinks = 3
)

var linkBlockTags = map[atom.Atom]bool{
	atom.Ul: true, atom.Ol: true, atom.Dl: true, atom.Div: true, atom.Section: true,
	atom.Table: true, atom.Header: true,
}

// pruneLinkHeavyBlocks removes menus, language pickers, tag clouds, and file
// listings: block containers whose text is mostly links. Content containers
// (<main>, <article>, <body>) are never removed themselves.
func pruneLinkHeavyBlocks(doc *html.Node) {
	stats := collectTextStats(doc)
	pageText := stats[doc].total
	var walk func(n *html.Node)
	walk = func(n *html.Node) {
		var next *html.Node
		for c := n.FirstChild; c != nil; c = next {
			next = c.NextSibling
			if c.Type != html.ElementNode {
				continue
			}
			s := stats[c]
			if linkBlockTags[c.DataAtom] && s.links >= linkHeavyMinLinks && s.linkDensity() >= linkHeavyDensity &&
				s.prose() < linkHeavyMaxProse && !containsPrimaryContent(c) &&
				2*s.total < pageText { // a block holding most of the page is the page, not a menu
				n.RemoveChild(c)
				continue
			}
			walk(c)
		}
	}
	walk(doc)
}

// Paragraph-like blocks whose text marks the surrounding container as content.
var proseTags = map[atom.Atom]bool{
	atom.P: true, atom.Pre: true, atom.Blockquote: true, atom.Li: true, atom.Td: true,
	atom.H1: true, atom.H2: true, atom.H3: true,
}

// minProseBlock is the shortest block that counts as a paragraph of prose.
const minProseBlock = 25

// specificShare is how much of the reference container's prose a nested
// candidate must hold to be preferred over it (it drops the wrapper's chrome).
const specificShare = 0.75

// findMainContent locates the primary content node. Candidates are semantic
// containers (<main>, <article>, role="main", common content ids) plus the
// container that holds the most paragraph text (Readability-style). Starting
// from the semantic container with the most prose (or <body> without one), it
// picks the most deeply nested candidate that still holds most of that prose,
// so wrappers full of chrome lose to the element holding the article itself.
func findMainContent(doc *html.Node) *html.Node {
	stats := collectTextStats(doc)
	var semantic []*html.Node
	var bodyNode *html.Node
	paragraphScore := make(map[*html.Node]int)
	depth := make(map[*html.Node]int)

	var walk func(n *html.Node, d int)
	walk = func(n *html.Node, d int) {
		if n.Type == html.ElementNode {
			depth[n] = d
			if n.DataAtom == atom.Body {
				bodyNode = n
			}
			role := strings.ToLower(getAttr(n, "role"))
			id := strings.ToLower(getAttr(n, "id"))
			if n.DataAtom == atom.Main || n.DataAtom == atom.Article ||
				role == "main" || id == "main" || id == "content" || id == "main-content" {
				semantic = append(semantic, n)
			}
			if proseTags[n.DataAtom] && n.Parent != nil {
				if prose := stats[n].prose(); prose >= minProseBlock {
					paragraphScore[n.Parent] += prose
					if gp := n.Parent.Parent; gp != nil {
						paragraphScore[gp] += prose / 2
					}
				}
			}
		}
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			walk(c, d+1)
		}
	}
	walk(doc, 0)

	fallback := bodyNode
	if fallback == nil {
		fallback = doc
	}

	// Reference container: the semantic candidate with the most prose.
	var reference *html.Node
	for _, cand := range semantic {
		if reference == nil || stats[cand].prose() > stats[reference].prose() {
			reference = cand
		}
	}
	if reference == nil || stats[reference].prose() <= 100 {
		reference = fallback
	}

	candidates := append([]*html.Node(nil), semantic...)
	var topParagraphNode *html.Node
	for node, score := range paragraphScore {
		if node.Type != html.ElementNode {
			continue
		}
		if topParagraphNode == nil || score > paragraphScore[topParagraphNode] ||
			(score == paragraphScore[topParagraphNode] && depth[node] > depth[topParagraphNode]) {
			topParagraphNode = node
		}
	}
	if topParagraphNode != nil {
		candidates = append(candidates, topParagraphNode)
	}

	refProse := stats[reference].prose()
	best := reference
	for _, cand := range candidates {
		if cand == reference || !isDescendant(cand, reference) {
			continue
		}
		if float64(stats[cand].prose()) >= specificShare*float64(refProse) && depth[cand] > depth[best] {
			best = cand
		}
	}
	return best
}

func isDescendant(n, ancestor *html.Node) bool {
	for p := n.Parent; p != nil; p = p.Parent {
		if p == ancestor {
			return true
		}
	}
	return false
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

// countText returns the number of visible characters in the tree.
func countText(root *html.Node) int {
	return collectTextStats(root)[root].total
}

// pruneDocument removes scripts, page chrome, and noise from doc, and with
// dropLinkLists also blocks that are mostly links. An element holding most of
// the page's text is the page itself, whatever its class or role says (for
// example a "sidebar-container" layout wrapper), so it is never removed.
func pruneDocument(doc *html.Node, dropLinkLists bool) {
	stats := collectTextStats(doc)
	pageText := stats[doc].total
	holdsMostText := func(n *html.Node) bool { return pageText > 0 && 2*stats[n].total >= pageText }
	cleanNode(doc, false, holdsMostText)
	if dropLinkLists {
		pruneLinkHeavyBlocks(doc)
	}
}
