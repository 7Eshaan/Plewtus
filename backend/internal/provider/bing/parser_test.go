package bing

import (
	"os"
	"strings"
	"testing"
)

func TestParseBingFixture(t *testing.T) {
	data, err := os.ReadFile("../../../testdata/fixtures/bing.html")
	if err != nil {
		t.Fatalf("failed to read Bing fixture: %v", err)
	}

	results, err := ParseHTMLResponse(strings.NewReader(string(data)), 10)
	if err != nil {
		t.Fatalf("ParseHTMLResponse failed: %v", err)
	}

	if len(results) == 0 {
		t.Fatalf("expected non-zero results from Bing fixture, got 0")
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

	t.Logf("First Bing result: title=%q url=%q snippet=%q", first.Title, first.URL, first.Snippet)

	// Check that URLs are cleanly unwrapped and not remaining as bing.com/ck/a
	for i, res := range results {
		if strings.Contains(res.URL, "bing.com/ck/a") {
			t.Errorf("[%d] URL was not unwrapped from bing click tracker: %s", i, res.URL)
		}
	}
}

func TestParseHTMLResponseIgnoresResultsOutsideOrganicContainer(t *testing.T) {
	html := `<html><body>
		<li class="b_algo"><h2><a href="https://spam.example">Sidebar Result</a></h2></li>
		<ol id="b_results"><li class="b_algo"><h2><a href="https://rust-lang.org">Rust</a></h2></li></ol>
	</body></html>`
	results, err := ParseHTMLResponse(strings.NewReader(html), 10)
	if err != nil {
		t.Fatalf("ParseHTMLResponse failed: %v", err)
	}
	if len(results) != 1 || results[0].Title != "Rust" {
		t.Fatalf("expected only the organic result, got %+v", results)
	}
}

func TestParseBingChallenge(t *testing.T) {
	challengeHTML := `<html><body>Our systems have detected unusual traffic from your computer network.</body></html>`
	_, err := ParseHTMLResponse(strings.NewReader(challengeHTML), 10)
	if err != ErrChallengeEncountered {
		t.Errorf("expected ErrChallengeEncountered, got %v", err)
	}
}
