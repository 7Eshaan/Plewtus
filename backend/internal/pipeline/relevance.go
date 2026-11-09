package pipeline

import (
	"math"
	"strings"
	"unicode"
)

var relevanceStopWords = map[string]struct{}{
	"a": {}, "an": {}, "and": {}, "are": {}, "as": {}, "at": {}, "be": {}, "because": {}, "been": {}, "being": {}, "but": {}, "by": {},
	"could": {}, "did": {}, "do": {}, "does": {}, "doing": {}, "for": {}, "from": {}, "had": {}, "has": {}, "have": {}, "having": {},
	"he": {}, "her": {}, "hers": {}, "him": {}, "his": {}, "how": {}, "i": {}, "if": {}, "in": {}, "into": {}, "is": {}, "it": {}, "its": {},
	"may": {}, "me": {}, "might": {}, "more": {}, "most": {}, "must": {}, "my": {}, "of": {}, "off": {}, "on": {}, "once": {}, "only": {}, "or": {}, "other": {}, "our": {}, "ours": {}, "out": {}, "over": {}, "own": {},
	"same": {}, "she": {}, "should": {}, "so": {}, "some": {}, "such": {}, "than": {}, "that": {}, "the": {}, "their": {}, "theirs": {}, "them": {}, "then": {}, "there": {}, "these": {}, "they": {}, "this": {}, "those": {}, "through": {}, "to": {}, "too": {}, "under": {}, "until": {}, "up": {}, "very": {}, "was": {}, "we": {}, "were": {}, "what": {}, "when": {}, "where": {}, "which": {}, "while": {}, "who": {}, "whom": {}, "why": {}, "will": {}, "with": {}, "would": {}, "you": {}, "your": {},
}

// queryProfile caches the tokenized form of a query so that scoring many
// candidates does not re-tokenize the query for every comparison.
type queryProfile struct {
	terms  map[string]struct{}
	phrase string
}

func newQueryProfile(query string) queryProfile {
	return queryProfile{terms: meaningfulTokens(query), phrase: normalizedPhrase(query)}
}

func (q queryProfile) hasTerms() bool { return len(q.terms) > 0 }

// QueryRelevance scores a result using query coverage, title matches, and exact
// phrase matches. The threshold is deliberately coverage based: long queries
// should not pass because of two incidental words, while exact phrases can
// survive providers that omit some query terms from their snippets.
func QueryRelevance(query, title, snippet string) (score float64, relevant bool) {
	return newQueryProfile(query).relevance(title, snippet)
}

func (q queryProfile) relevance(title, snippet string) (score float64, relevant bool) {
	queryTerms := q.terms
	if len(queryTerms) == 0 {
		return 1, true
	}

	titleTokens := tokenSet(title)
	snippetTokens := tokenSet(snippet)
	matched := 0
	titleMatched := 0
	snippetMatched := 0
	missingCompoundIdentifier := false
	for term := range queryTerms {
		_, inTitle := titleTokens[term]
		_, inSnippet := snippetTokens[term]
		if inSnippet {
			snippetMatched++
		}
		if inTitle || inSnippet {
			matched++
			if inTitle {
				titleMatched++
			}
		} else if isCompoundIdentifier(term) {
			// Punctuation distinguishes identifiers such as C++ and C#. A
			// generic C result must not qualify for one of those queries merely
			// because the other query terms happen to match.
			missingCompoundIdentifier = true
		}
	}
	n := float64(len(queryTerms))
	coverage := float64(matched) / n
	titleCoverage := float64(titleMatched) / n
	snippetCoverage := float64(snippetMatched) / n
	phraseMatch := q.containsPhrase(title) || q.containsPhrase(snippet)
	score = 0.55*coverage + 0.30*titleCoverage + 0.15*snippetCoverage
	if phraseMatch {
		score = math.Min(1, score+0.15)
	}

	minimumCoverage := 1
	if len(queryTerms) > 1 {
		minimumCoverage = int(math.Ceil(n * 0.60))
		if minimumCoverage < 2 {
			minimumCoverage = 2
		}
	}
	return score, !missingCompoundIdentifier && (phraseMatch || matched >= minimumCoverage)
}

func (q queryProfile) containsPhrase(text string) bool {
	if q.phrase == "" {
		return false
	}
	return strings.Contains(" "+normalizedPhrase(text)+" ", " "+q.phrase+" ")
}

// selectTitle keeps whichever title better matches the query, falling back to
// the more complete (non-truncated, longer) title on ties.
func (q queryProfile) selectTitle(curr, candidate string) string {
	if !q.hasTerms() {
		return selectBetterTitle(curr, candidate)
	}
	currentScore, _ := q.relevance(curr, "")
	candidateScore, _ := q.relevance(candidate, "")
	if candidateScore > currentScore+1e-9 {
		return strings.TrimSpace(candidate)
	}
	if currentScore > candidateScore+1e-9 {
		return strings.TrimSpace(curr)
	}
	return selectBetterTitle(curr, candidate)
}

// selectSnippet keeps whichever snippet better matches the query.
func (q queryProfile) selectSnippet(curr, candidate string) string {
	if !q.hasTerms() {
		return selectBetterSnippet(curr, candidate)
	}
	currentScore, _ := q.relevance("", curr)
	candidateScore, _ := q.relevance("", candidate)
	if candidateScore > currentScore+1e-9 {
		return cleanSnippet(candidate)
	}
	if currentScore > candidateScore+1e-9 {
		return cleanSnippet(curr)
	}
	return selectBetterSnippet(curr, candidate)
}

func isCompoundIdentifier(term string) bool {
	switch term {
	case "cplusplus", "csharp", "fsharp", "dotnet":
		return true
	default:
		return false
	}
}

func normalizedPhrase(value string) string {
	value = strings.ToLower(value)
	for _, replacement := range [][2]string{{"c++", "cplusplus"}, {"c#", "csharp"}, {"f#", "fsharp"}, {".net", "dotnet"}} {
		value = strings.ReplaceAll(value, replacement[0], replacement[1])
	}
	var out strings.Builder
	space := true
	for _, r := range value {
		if isCJK(r) {
			if !space {
				out.WriteByte(' ')
			}
			out.WriteRune(r)
			out.WriteByte(' ')
			space = true
		} else if unicode.IsLetter(r) || unicode.IsNumber(r) {
			out.WriteRune(r)
			space = false
		} else if !space {
			out.WriteByte(' ')
			space = true
		}
	}
	return strings.TrimSpace(out.String())
}

func meaningfulTokens(value string) map[string]struct{} {
	tokens := tokenSet(value)
	for token := range tokens {
		if _, stop := relevanceStopWords[token]; stop {
			delete(tokens, token)
		}
	}
	// Keep all-stopword queries usable instead of making them match everything.
	if len(tokens) == 0 {
		return tokenSet(value)
	}
	return tokens
}

func tokenSet(value string) map[string]struct{} {
	tokens := make(map[string]struct{})
	value = strings.ToLower(value)
	// Preserve common programming-language identifiers whose punctuation is
	// meaningful to search (plain tokenization would reduce C++ to the common
	// single-letter token "c").
	for _, replacement := range [][2]string{{"c++", "cplusplus"}, {"c#", "csharp"}, {"f#", "fsharp"}, {".net", "dotnet"}} {
		value = strings.ReplaceAll(value, replacement[0], replacement[1])
	}
	var token strings.Builder
	flush := func() {
		if token.Len() == 0 {
			return
		}
		normalized := token.String()
		if normalized == "golang" {
			normalized = "go"
		}
		tokens[normalized] = struct{}{}
		token.Reset()
	}
	for _, r := range value {
		switch {
		case isCJK(r):
			// CJK scripts do not separate words with spaces; index each
			// character so queries can match inside longer runs of text.
			flush()
			tokens[string(r)] = struct{}{}
		case unicode.IsLetter(r) || unicode.IsNumber(r):
			token.WriteRune(r)
		default:
			flush()
		}
	}
	flush()
	return tokens
}

func isCJK(r rune) bool {
	return unicode.In(r, unicode.Han, unicode.Hiragana, unicode.Katakana, unicode.Hangul)
}
