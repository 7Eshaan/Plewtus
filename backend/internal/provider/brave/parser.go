package brave

import (
	"encoding/json"
	"io"
	"strings"

	"golang.org/x/net/html"
	"golang.org/x/net/html/atom"

	"plewtus/internal/model"
	"plewtus/internal/provider/htmlutil"
)

// ParseHTMLResponse extracts organic results from a Brave results page.
//
// Each organic result is a `div.snippet[data-type="web"]`. Its main link is
// the first anchor of `.result-content`; the full title is the `title`
// attribute of `.title`; the snippet is `.generic-snippet .content`, which may
// start with a `span.t-secondary` date ("19 August 2026 -"). Video and news
// clusters use other data-type values, and site links nested inside a web
// result (`.result-cluster`) are skipped.
func ParseHTMLResponse(r io.Reader) ([]model.RawSearchResult, error) {
	doc, err := html.Parse(r)
	if err != nil {
		return nil, err
	}

	nodes := htmlutil.FindAll(doc, func(n *html.Node) bool {
		if n.Type != html.ElementNode || n.Data != "div" || !htmlutil.HasClass(n, "snippet") {
			return false
		}
		t, _ := htmlutil.GetAttr(n, "data-type")
		return t == "web"
	})

	var results []model.RawSearchResult
	for _, node := range nodes {
		content := htmlutil.FindOne(node, func(n *html.Node) bool {
			return n.Type == html.ElementNode && htmlutil.HasClass(n, "result-content")
		})
		if content == nil {
			content = node
		}
		link := htmlutil.FindOne(content, func(n *html.Node) bool {
			if n.Type != html.ElementNode || n.Data != "a" || insideCluster(n, content) {
				return false
			}
			href, _ := htmlutil.GetAttr(n, "href")
			return strings.HasPrefix(href, "http://") || strings.HasPrefix(href, "https://")
		})
		if link == nil {
			continue
		}
		href, _ := htmlutil.GetAttr(link, "href")

		title := ""
		if titleNode := htmlutil.FindOne(link, func(n *html.Node) bool {
			return n.Type == html.ElementNode && htmlutil.HasClass(n, "title")
		}); titleNode != nil {
			// The attribute holds the untruncated title.
			if full, ok := htmlutil.GetAttr(titleNode, "title"); ok {
				title = strings.Join(strings.Fields(full), " ")
			}
			if title == "" {
				title = htmlutil.TextContent(titleNode)
			}
		}
		if title == "" {
			continue
		}

		snippet, published := "", ""
		if snippetNode := htmlutil.FindOne(content, func(n *html.Node) bool {
			return n.Type == html.ElementNode && htmlutil.HasClass(n, "content") && hasAncestorClass(n, "generic-snippet", content)
		}); snippetNode != nil {
			snippet = htmlutil.TextContent(snippetNode)
			if d := dateNode(snippetNode); d != nil {
				dateText := htmlutil.TextContent(d)
				published = strings.TrimSpace(strings.TrimRight(dateText, " -–·"))
				snippet = strings.TrimSpace(strings.TrimPrefix(snippet, dateText))
			}
		}

		results = append(results, model.RawSearchResult{
			Title:         title,
			URL:           href,
			Snippet:       snippet,
			Position:      len(results) + 1,
			PublishedDate: published,
		})
	}
	return results, nil
}

// dateNode returns the leading date span of a snippet, if any.
func dateNode(snippet *html.Node) *html.Node {
	return htmlutil.FindOne(snippet, func(n *html.Node) bool {
		return n.Type == html.ElementNode && n.Data == "span" && htmlutil.HasClass(n, "t-secondary")
	})
}

// insideCluster reports whether n sits in a `.result-cluster` (site links)
// below stop.
func insideCluster(n, stop *html.Node) bool {
	return hasAncestorClass(n, "result-cluster", stop)
}

func hasAncestorClass(n *html.Node, class string, stop *html.Node) bool {
	for p := n.Parent; p != nil && p != stop; p = p.Parent {
		if htmlutil.HasClass(p, class) {
			return true
		}
	}
	return false
}

// apiResponse is the subset of the Search API response used here.
type apiResponse struct {
	Web struct {
		Results []struct {
			Title       string `json:"title"`
			URL         string `json:"url"`
			Description string `json:"description"`
			Age         string `json:"age"`
			PageAge     string `json:"page_age"`
		} `json:"results"`
	} `json:"web"`
}

// ParseAPIResponse decodes a Search API response. Descriptions carry
// <strong> highlight tags, which are stripped.
func ParseAPIResponse(body []byte) ([]model.RawSearchResult, error) {
	var parsed apiResponse
	if err := json.Unmarshal(body, &parsed); err != nil {
		return nil, err
	}

	results := make([]model.RawSearchResult, 0, len(parsed.Web.Results))
	for _, r := range parsed.Web.Results {
		link := strings.TrimSpace(r.URL)
		if !strings.HasPrefix(link, "http://") && !strings.HasPrefix(link, "https://") {
			continue
		}
		title := stripTags(r.Title)
		if title == "" {
			title = link
		}
		published := r.PageAge
		if published == "" {
			published = r.Age
		}
		results = append(results, model.RawSearchResult{
			Title:         title,
			URL:           link,
			Snippet:       stripTags(r.Description),
			Position:      len(results) + 1,
			PublishedDate: published,
		})
	}
	return results, nil
}

// stripTags turns an HTML fragment into plain, whitespace-normalised text.
func stripTags(fragment string) string {
	nodes, err := html.ParseFragment(strings.NewReader(fragment), &html.Node{Type: html.ElementNode, Data: "div", DataAtom: atom.Div})
	if err != nil {
		return strings.Join(strings.Fields(fragment), " ")
	}
	var sb strings.Builder
	for _, n := range nodes {
		sb.WriteString(htmlutil.TextContent(n))
		sb.WriteString(" ")
	}
	return strings.Join(strings.Fields(sb.String()), " ")
}
