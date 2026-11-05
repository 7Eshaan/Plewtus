package htmlutil

import (
	"encoding/base64"
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

// HasAncestorID reports whether a node is nested beneath an element with the
// requested id. Providers use this to keep extraction inside the organic
// results container instead of matching similarly-classed page modules.
func HasAncestorID(n *html.Node, id string) bool {
	for ancestor := n.Parent; ancestor != nil; ancestor = ancestor.Parent {
		if ancestorID, ok := GetAttr(ancestor, "id"); ok && ancestorID == id {
			return true
		}
	}
	return false
}

// FindAllInAncestorScope prefers nodes nested under the expected organic
// results container. Some provider responses omit or rename that container, so
// it falls back to the selector across the document when the scoped search is
// empty. Query relevance filtering is applied later in the pipeline.
func FindAllInAncestorScope(n *html.Node, id string, predicate func(*html.Node) bool) []*html.Node {
	scoped := FindAll(n, func(candidate *html.Node) bool {
		return predicate(candidate) && HasAncestorID(candidate, id)
	})
	if len(scoped) > 0 {
		return scoped
	}
	return FindAll(n, predicate)
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

// UnwrapURL cleans and unwraps redirect URLs across search engines
// (DuckDuckGo uddg, Bing /ck/a u=a1..., Yahoo /RU=...).
func UnwrapURL(rawURL string) string {
	trimmed := strings.TrimSpace(rawURL)
	if trimmed == "" {
		return ""
	}

	// Protocol-relative URLs
	if strings.HasPrefix(trimmed, "//") {
		trimmed = "https:" + trimmed
	}

	// 1. Yahoo redirect unwrapping: .../RU=<encoded_url>/RK=...
	if strings.Contains(trimmed, "search.yahoo.com") && strings.Contains(trimmed, "/RU=") {
		parts := strings.Split(trimmed, "/RU=")
		if len(parts) > 1 {
			targetPart := parts[1]
			if idx := strings.Index(targetPart, "/RK="); idx != -1 {
				targetPart = targetPart[:idx]
			} else if idx := strings.Index(targetPart, "/"); idx != -1 {
				targetPart = targetPart[:idx]
			}
			if unescaped, err := url.QueryUnescape(targetPart); err == nil && strings.HasPrefix(unescaped, "http") {
				return unescaped
			}
		}
	}

	parsed, err := url.Parse(trimmed)
	if err != nil {
		return trimmed
	}

	q := parsed.Query()

	// 2. DuckDuckGo redirect unwrapping: uddg=<url>
	if uddg := q.Get("uddg"); uddg != "" {
		if unescaped, err := url.QueryUnescape(uddg); err == nil && unescaped != "" {
			return unescaped
		}
		return uddg
	}

	// 3. Bing redirect unwrapping: /ck/a?!...&u=a1<base64>...
	if strings.Contains(parsed.Host, "bing.com") && strings.Contains(parsed.Path, "/ck/a") {
		if uVal := q.Get("u"); uVal != "" {
			if decoded := decodeBingU(uVal); decoded != "" && strings.HasPrefix(decoded, "http") {
				return decoded
			}
		}
	}

	return trimmed
}

func decodeBingU(uVal string) string {
	if len(uVal) < 3 {
		return ""
	}
	// Bing prefixes base64 string with a1 or a0
	b64 := uVal
	if strings.HasPrefix(uVal, "a1") || strings.HasPrefix(uVal, "a0") {
		b64 = uVal[2:]
	}

	// Add padding if missing
	if pad := len(b64) % 4; pad > 0 {
		b64 += strings.Repeat("=", 4-pad)
	}

	if decoded, err := base64.URLEncoding.DecodeString(b64); err == nil {
		return string(decoded)
	}
	if decoded, err := base64.RawURLEncoding.DecodeString(strings.TrimRight(b64, "=")); err == nil {
		return string(decoded)
	}
	if decoded, err := base64.StdEncoding.DecodeString(b64); err == nil {
		return string(decoded)
	}

	return ""
}
