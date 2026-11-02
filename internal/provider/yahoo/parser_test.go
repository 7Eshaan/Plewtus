package yahoo

import (
	"os"
	"strings"
	"testing"
)

func TestParseYahooFixture(t *testing.T) {
	data, err := os.ReadFile("../../../testdata/fixtures/yahoo.html")
	if err != nil {
		t.Fatalf("failed to read Yahoo fixture: %v", err)
	}

	results, err := ParseHTMLResponse(strings.NewReader(string(data)), 10)
	if err != nil {
		t.Fatalf("ParseHTMLResponse failed: %v", err)
	}

	if len(results) == 0 {
		t.Fatalf("expected non-zero results from Yahoo fixture, got 0")
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

	t.Logf("First Yahoo result: title=%q url=%q snippet=%q", first.Title, first.URL, first.Snippet)

	// Ensure no Yahoo redirect click tracking remains
	for i, res := range results {
		if strings.Contains(res.URL, "r.search.yahoo.com") {
			t.Errorf("[%d] URL was not unwrapped from yahoo redirect: %s", i, res.URL)
		}
	}
}

func TestParseYahooChallenge(t *testing.T) {
	challengeHTML := `<html><body>Robot Check: please verify you are human</body></html>`
	_, err := ParseHTMLResponse(strings.NewReader(challengeHTML), 10)
	if err != ErrChallengeEncountered {
		t.Errorf("expected ErrChallengeEncountered, got %v", err)
	}
}
