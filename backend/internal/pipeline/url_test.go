package pipeline

import (
	"testing"
)

func TestNormalizeURL_TrackingParameters(t *testing.T) {
	tests := []struct {
		name         string
		input        string
		expectedURL  string
		expectedKey  string
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
			name:        "Semantic query params preserved while tracking stripped",
			input:       "https://pkg.go.dev/net/http?utm_source=ddg&tab=doc&ref=newsletter",
			expectedURL: "https://pkg.go.dev/net/http?tab=doc",
			expectedKey: "pkg.go.dev/net/http?tab=doc",
		},
		{
			name:        "Query parameters deterministically sorted",
			input:       "https://example.com/search?z=3&a=1&m=2",
			expectedURL: "https://example.com/search?a=1&m=2&z=3",
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
	_, _, err := NormalizeURL("")
	if err == nil {
		t.Errorf("expected error for empty URL")
	}

	_, _, err = NormalizeURL("   ")
	if err == nil {
		t.Errorf("expected error for whitespace URL")
	}
}
