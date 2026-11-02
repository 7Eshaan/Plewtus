package htmlutil

import (
	"strings"
	"testing"

	"golang.org/x/net/html"
)

func TestHTMLUtilFunctions(t *testing.T) {
	docStr := `
	<div class="result web-result">
		<h2 class="title"><a class="link" href="//duckduckgo.com/l/?uddg=https%3A%2F%2Fexample.com%2Fdoc&rut=123">Example <span>Title</span></a></h2>
		<p class="snippet">This is an <b>example</b> snippet.</p>
	</div>
	<div class="result ad-result">Ad block</div>
	`

	doc, err := html.Parse(strings.NewReader(docStr))
	if err != nil {
		t.Fatalf("failed to parse test HTML: %v", err)
	}

	results := FindAll(doc, func(n *html.Node) bool {
		return n.Type == html.ElementNode && HasClass(n, "result")
	})

	if len(results) != 2 {
		t.Fatalf("expected 2 results, got %d", len(results))
	}

	linkNode := FindOne(results[0], func(n *html.Node) bool {
		return n.Type == html.ElementNode && n.Data == "a" && HasClass(n, "link")
	})
	if linkNode == nil {
		t.Fatalf("linkNode not found")
	}

	titleText := TextContent(linkNode)
	if titleText != "Example Title" {
		t.Errorf("expected 'Example Title', got %q", titleText)
	}

	href, ok := GetAttr(linkNode, "href")
	if !ok {
		t.Fatalf("href attribute not found")
	}

	unwrapped := UnwrapURL(href)
	if unwrapped != "https://example.com/doc" {
		t.Errorf("expected 'https://example.com/doc', got %q", unwrapped)
	}

	snippetNode := FindOne(results[0], func(n *html.Node) bool {
		return n.Type == html.ElementNode && HasClass(n, "snippet")
	})
	if snippetNode == nil {
		t.Fatalf("snippetNode not found")
	}

	snippetText := TextContent(snippetNode)
	if snippetText != "This is an example snippet." {
		t.Errorf("expected 'This is an example snippet.', got %q", snippetText)
	}
}
