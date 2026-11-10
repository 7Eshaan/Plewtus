package bing

import (
	"errors"
	"strings"
	"testing"
)

// Regression: a real result that quotes Bing's block-page wording must not be
// mistaken for a block page.
func TestParse_ResultQuotingBlockWordingIsNotAChallenge(t *testing.T) {
	page := `<html><body><ol id="b_results"><li class="b_algo"><h2><a href="https://support.example.com/kb/unusual-traffic">Why do I see "unusual traffic from your computer network"?</a></h2><div class="b_caption"><p>The message unusual traffic from your computer network appears when...</p></div></li></ol></body></html>`
	results, err := ParseHTMLResponse(strings.NewReader(page), 10)
	if err != nil || len(results) != 1 {
		t.Fatalf("expected 1 result and no error, got %d, %v", len(results), err)
	}
}

func TestParse_BlockPageWithoutResultsIsAChallenge(t *testing.T) {
	page := `<html><body><h1>Please verify you are a human</h1></body></html>`
	if _, err := ParseHTMLResponse(strings.NewReader(page), 10); !errors.Is(err, ErrChallengeEncountered) {
		t.Fatalf("expected ErrChallengeEncountered, got %v", err)
	}
}
