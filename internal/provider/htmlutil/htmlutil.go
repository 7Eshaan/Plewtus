package htmlutil

import (
	"net/url"
	"strings"

	"golang.org/x/net/html"
)

// FindAll traverses the HTML node tree and returns all nodes matching the predicate.
func FindAll(n *html.Node, predicate func(*html.Node) bool) []*html.Node {
	var matches []*html.Node
	var traverse func(*html.Node)
	traverse = func(curr *html.Node) {
		if curr == nil {
			return
		}
		if predicate(curr) {
			matches = append(matches, curr)
		}
		for c := curr.FirstChild; c != nil; c = c.NextSibling {
			traverse(c)
		}
	}
	traverse(n)
	return matches
}

// FindOne traverses the HTML node tree and returns the first node matching the predicate.
func FindOne(n *html.Node, predicate func(*html.Node) bool) *html.Node {
	if n == nil {
		return nil
	}
	if predicate(n) {
		return n
	}
	for c := n.FirstChild; c != nil; c = c.NextSibling {
		if match := FindOne(c, predicate); match != nil {
			return match
		}
	}
	return nil
}

// GetAttr returns the value of the given attribute if present.
func GetAttr(n *html.Node, key string) (string, bool) {
	if n == nil {
		return "", false
	}
	for _, attr := range n.Attr {
		if strings.EqualFold(attr.Key, key) {
			return attr.Val, true
		}
	}
	return "", false
}

// HasClass checks if the node's class attribute contains the given class name.
func HasClass(n *html.Node, className string) bool {
	val, ok := GetAttr(n, "class")
	if !ok {
		return false
	}
	classes := strings.Fields(val)
	for _, c := range classes {
		if c == className {
			return true
		}
	}
	return false
}

// HasAnyClass checks if the node contains any of the given class names.
func HasAnyClass(n *html.Node, classNames ...string) bool {
	val, ok := GetAttr(n, "class")
	if !ok {
		return false
	}
	classes := strings.Fields(val)
	for _, c := range classes {
		for _, target := range classNames {
			if c == target {
				return true
			}
		}
	}
	return false
}

// TextContent recursively extracts and concatenates all text under a node,
// normalizing whitespace.
func TextContent(n *html.Node) string {
	if n == nil {
		return ""
	}
	var sb strings.Builder
	var collect func(*html.Node)
	collect = func(curr *html.Node) {
		if curr == nil {
			return
		}
		if curr.Type == html.TextNode {
			sb.WriteString(curr.Data)
			sb.WriteString(" ")
		}
		for c := curr.FirstChild; c != nil; c = c.NextSibling {
			collect(c)
		}
	}
	collect(n)
	return strings.Join(strings.Fields(sb.String()), " ")
}

// UnwrapURL cleans and unwraps redirect URLs (like DuckDuckGo /l/?uddg=... or protocol-relative URLs).
func UnwrapURL(rawURL string) string {
	trimmed := strings.TrimSpace(rawURL)
	if trimmed == "" {
		return ""
	}

	// Handle protocol-relative URLs
	if strings.HasPrefix(trimmed, "//") {
		trimmed = "https:" + trimmed
	}

	parsed, err := url.Parse(trimmed)
	if err != nil {
		return trimmed
	}

	// Check if wrapped in a redirect parameter (e.g. uddg on DuckDuckGo, u3/url on others)
	q := parsed.Query()
	if uddg := q.Get("uddg"); uddg != "" {
		if unescaped, err := url.QueryUnescape(uddg); err == nil && unescaped != "" {
			return unescaped
		}
		return uddg
	}

	return trimmed
}
