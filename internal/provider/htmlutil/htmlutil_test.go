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

func TestUnwrapURL_SearchEngines(t *testing.T) {
	tests := []struct {
		name     string
		rawURL   string
		expected string
	}{
		{
			name:     "direct URL",
			rawURL:   "https://go.dev/doc/install",
			expected: "https://go.dev/doc/install",
		},
		{
			name:     "protocol-relative",
			rawURL:   "//go.dev/learn",
			expected: "https://go.dev/learn",
		},
		{
			name:     "DuckDuckGo uddg",
			rawURL:   "https://duckduckgo.com/l/?uddg=https%3A%2F%2Fpkg.go.dev%2F&rut=abc",
			expected: "https://pkg.go.dev/",
		},
		{
			name:     "Bing u=a1... base64",
			rawURL:   "https://www.bing.com/ck/a?!&&p=123&u=a1aHR0cHM6Ly9lbi53aWtpcGVkaWEub3JnL3dpa2kvR29fKHByb2dyYW1taW5nX2xhbmd1YWdlKQ&ntb=1",
			expected: "https://en.wikipedia.org/wiki/Go_(programming_language)",
		},
		{
			name:     "Bing direct go.dev",
			rawURL:   "https://www.bing.com/ck/a?!&&p=456&u=a1aHR0cHM6Ly9nby5kZXYv&ntb=1",
			expected: "https://go.dev/",
		},
		{
			name:     "Yahoo /RU=.../RK=...",
			rawURL:   "https://r.search.yahoo.com/_ylt=A2RTMBLAeuhqywIA0IhXNyoA;_ylu=.../RV=2/RE=1794830273/RO=10/RU=https%3a%2f%2fgo.dev%2f/RK=2/RS=sf5qfStkBLTqVpqiRHCppTZ_tgk-",
			expected: "https://go.dev/",
		},
		{
			name:     "Yahoo GeeksforGeeks",
			rawURL:   "https://r.search.yahoo.com/_ylt=.../RU=https%3a%2f%2fwww.geeksforgeeks.org%2fblogs%2f100-days-of-golang%2f/RK=2/RS=...",
			expected: "https://www.geeksforgeeks.org/blogs/100-days-of-golang/",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			actual := UnwrapURL(tc.rawURL)
			if actual != tc.expected {
				t.Errorf("UnwrapURL(%q) = %q, expected %q", tc.rawURL, actual, tc.expected)
			}
		})
	}
}
