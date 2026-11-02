package duckduckgo

import (
	"os"
	"strings"
	"testing"
)

func TestParseHTMLFixture(t *testing.T) {
	data, err := os.ReadFile("../../../testdata/fixtures/duckduckgo_html.html")
	if err != nil {
		t.Fatalf("failed to read HTML fixture: %v", err)
	}

	results, err := ParseHTMLResponse(strings.NewReader(string(data)), 10)
	if err != nil {
		t.Fatalf("ParseHTMLResponse failed: %v", err)
	}

	if len(results) == 0 {
		t.Fatalf("expected non-zero results from HTML fixture, got 0")
	}

	first := results[0]
	if first.Title == "" {
		t.Errorf("expected non-empty title for first result")
	}
	if !strings.HasPrefix(first.URL, "https://") && !strings.HasPrefix(first.URL, "http://") {
		t.Errorf("expected valid HTTP/HTTPS URL, got %q", first.URL)
	}
	if first.Snippet == "" {
		t.Errorf("expected non-empty snippet for first result")
	}
	if first.Position != 1 {
		t.Errorf("expected position 1, got %d", first.Position)
	}

	// Verify ad filtering: no results should have duckduckgo.com/y.js or bing ad click URLs
	for _, res := range results {
		if strings.Contains(res.URL, "duckduckgo.com/y.js") || strings.Contains(res.URL, "aclick") {
			t.Errorf("ad URL leaked into organic results: %s", res.URL)
		}
	}
}

func TestParseLiteFixture(t *testing.T) {
	data, err := os.ReadFile("../../../testdata/fixtures/duckduckgo_lite.html")
	if err != nil {
		t.Fatalf("failed to read Lite fixture: %v", err)
	}

	results, err := ParseLiteResponse(strings.NewReader(string(data)), 10)
	if err != nil {
		t.Fatalf("ParseLiteResponse failed: %v", err)
	}

	if len(results) == 0 {
		t.Fatalf("expected non-zero results from Lite fixture, got 0")
	}

	first := results[0]
	if first.Title == "" {
		t.Errorf("expected non-empty title for first result")
	}
	if !strings.HasPrefix(first.URL, "https://") && !strings.HasPrefix(first.URL, "http://") {
		t.Errorf("expected valid HTTP/HTTPS URL, got %q", first.URL)
	}
	if first.Snippet == "" {
		t.Errorf("expected non-empty snippet for first result")
	}
	if first.Position != 1 {
		t.Errorf("expected position 1, got %d", first.Position)
	}

	// Verify ad filtering
	for _, res := range results {
		if strings.Contains(res.URL, "duckduckgo.com/y.js") || strings.Contains(res.URL, "aclick") {
			t.Errorf("ad URL leaked into organic results: %s", res.URL)
		}
	}
}

func TestParseChallengeEncountered(t *testing.T) {
	challengeHTML := `
	<html>
	<body>
	<div class="anomaly-modal__title">Unfortunately, bots use DuckDuckGo too.</div>
	</body>
	</html>
	`

	_, err := ParseHTMLResponse(strings.NewReader(challengeHTML), 10)
	if err != ErrChallengeEncountered {
		t.Errorf("expected ErrChallengeEncountered, got %v", err)
	}

	_, err = ParseLiteResponse(strings.NewReader(challengeHTML), 10)
	if err != ErrChallengeEncountered {
		t.Errorf("expected ErrChallengeEncountered, got %v", err)
	}
}
