package pipeline

import (
	"testing"
)

func TestNormalizeURL_TrackingParameters(t *testing.T) {
	tests := []struct {
		name        string
		input       string
		expectedURL string
		expectedKey string
	}{
		{
			name:        "UTM parameters removed",
			input:       "https://example.com/article?utm_source=twitter&utm_medium=social&utm_campaign=launch",
			expectedURL: "https://example.com/article",
			expectedKey: "example.com/article",
		},
		{
			name:        "Click IDs removed (gclid, fbclid, msclkid)",
			input:       "https://example.com/products/shoes?gclid=12345&fbclid=abcdef&msclkid=xyz",
			expectedURL: "https://example.com/products/shoes",
			expectedKey: "example.com/products/shoes",
		},
		{
			name:        "Semantic query params preserved while tracking stripped; referral params only ignored for matching",
			input:       "https://pkg.go.dev/net/http?utm_source=ddg&tab=doc&ref=newsletter",
			expectedURL: "https://pkg.go.dev/net/http?tab=doc&ref=newsletter",
			expectedKey: "pkg.go.dev/net/http?tab=doc",
		},
		{
			name:        "Query order kept for display, sorted for matching",
			input:       "https://example.com/search?z=3&a=1&m=2",
			expectedURL: "https://example.com/search?z=3&a=1&m=2",
			expectedKey: "example.com/search?a=1&m=2&z=3",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			clean, key, err := NormalizeURL(tc.input)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if clean != tc.expectedURL {
				t.Errorf("clean URL: expected %q, got %q", tc.expectedURL, clean)
			}
			if key != tc.expectedKey {
				t.Errorf("canonical key: expected %q, got %q", tc.expectedKey, key)
			}
		})
	}
}

func TestNormalizeURL_CanonicalMatching(t *testing.T) {
	// All variations in a group should produce the exact same canonicalKey
	groups := []struct {
		groupName string
		urls      []string
	}{
		{
			groupName: "Home page variations",
			urls: []string{
				"https://www.example.com/",
				"http://example.com",
				"http://www.example.com/index.html",
				"https://example.com/#top",
				"https://www.example.com/?utm_source=feed",
			},
		},
		{
			groupName: "Article page variations",
			urls: []string{
				"https://example.com/blog/go-concurrency/",
				"https://example.com/blog/go-concurrency",
				"http://www.example.com/blog/go-concurrency#section-2",
				"https://www.example.com/blog/go-concurrency?fbclid=xyz",
			},
		},
		{
			groupName: "Protocol-relative and ports",
			urls: []string{
				"//golang.org/doc/",
				"https://golang.org:443/doc",
				"http://golang.org:80/doc/",
			},
		},
	}

	for _, g := range groups {
		t.Run(g.groupName, func(t *testing.T) {
			var firstKey string
			for i, raw := range g.urls {
				clean, key, err := NormalizeURL(raw)
				if err != nil {
					t.Fatalf("failed normalizing %q: %v", raw, err)
				}
				if i == 0 {
					firstKey = key
				} else if key != firstKey {
					t.Errorf("URL %q key %q does not match expected %q (clean: %q)", raw, key, firstKey, clean)
				}
			}
		})
	}
}

func TestNormalizeURL_InvalidOrEmpty(t *testing.T) {
	for _, raw := range []string{"", "   ", "not a url ://", "https://user:pass@example.com/"} {
		if _, _, err := NormalizeURL(raw); err == nil {
			t.Errorf("expected error for invalid URL %q", raw)
		}
	}
}

func TestNormalizeURL_PreservesCaseSensitivePathAndQuery(t *testing.T) {
	_, pathUpper, err := NormalizeURL("https://Example.com/Docs/API?ID=ABC")
	if err != nil {
		t.Fatal(err)
	}
	_, pathLower, err := NormalizeURL("https://example.com/docs/api?ID=ABC")
	if err != nil {
		t.Fatal(err)
	}
	_, queryLower, err := NormalizeURL("https://example.com/Docs/API?ID=abc")
	if err != nil {
		t.Fatal(err)
	}
	if pathUpper == pathLower || pathUpper == queryLower {
		t.Fatalf("case-sensitive path/query variants were merged: %q %q %q", pathUpper, pathLower, queryLower)
	}
}

// Regression: the displayed URL must still open the same page.
func TestNormalizeURL_DisplayURLIsNotRewritten(t *testing.T) {
	for in, want := range map[string]string{
		"https://example.com/a%2Fb/c":               "https://example.com/a%2Fb/c",
		"https://example.com/page?v":                "https://example.com/page?v",
		"https://example.com/search?q=a%20b&z=1":    "https://example.com/search?q=a%20b&z=1",
		"https://example.com/docs/":                 "https://example.com/docs/",
		"https://github.com/o/r/blob/x?ref=main":    "https://github.com/o/r/blob/x?ref=main",
		"HTTP://Example.com:80/Index.HTML#frag":     "http://example.com/Index.HTML",
		"https://example.com/x?utm_source=a&id=7#y": "https://example.com/x?id=7",
		"https://en.wikipedia.org/wiki/C++":         "https://en.wikipedia.org/wiki/C++",
	} {
		got, _, err := NormalizeURL(in)
		if err != nil || got != want {
			t.Errorf("NormalizeURL(%q) = %q, %v; want %q", in, got, err, want)
		}
	}
}
