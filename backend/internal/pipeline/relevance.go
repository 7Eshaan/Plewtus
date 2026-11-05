package pipeline

import (
	"math"
	"strings"
	"unicode"
)

var relevanceStopWords = map[string]struct{}{
	"a": {}, "an": {}, "and": {}, "are": {}, "as": {}, "at": {}, "be": {}, "by": {},
	"for": {}, "from": {}, "how": {}, "in": {}, "into": {}, "is": {}, "it": {}, "of": {},
	"on": {}, "or": {}, "the": {}, "to": {}, "was": {}, "what": {}, "when": {}, "where": {},
	"which": {}, "who": {}, "why": {}, "with": {},
}

// QueryRelevance estimates whether a result has meaningful lexical overlap with
// the query. It is a conservative guard against unrelated or polluted SERPs,
// not a semantic search model. A result must match at least half of the
// significant query terms (and at least two terms for multi-term queries).
func QueryRelevance(query, title, snippet string) (score float64, relevant bool) {
	queryTerms := meaningfulTokens(query)
	if len(queryTerms) == 0 {
		return 1, true
	}

	titleTokens := tokenSet(title)
	contentTokens := tokenSet(title + " " + snippet)
	matched := 0
	titleMatched := 0
	for term := range queryTerms {
		if _, ok := contentTokens[term]; ok {
			matched++
			if _, inTitle := titleTokens[term]; inTitle {
				titleMatched++
			}
		}
	}

	minimumMatches := int(math.Ceil(float64(len(queryTerms)) * 0.5))
	if len(queryTerms) > 1 && minimumMatches < 2 {
		minimumMatches = 2
	}
	if minimumMatches > len(queryTerms) {
		minimumMatches = len(queryTerms)
	}

	coverage := float64(matched) / float64(len(queryTerms))
	score = coverage + 0.15*float64(titleMatched)/float64(len(queryTerms))
	if score > 1 {
		score = 1
	}
	return score, matched >= minimumMatches
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
	var token strings.Builder
	flush := func() {
		if token.Len() == 0 {
			return
		}
		value := strings.ToLower(token.String())
		if value == "golang" {
			value = "go"
		}
		tokens[value] = struct{}{}
		token.Reset()
	}
	for _, r := range value {
		if unicode.IsLetter(r) || unicode.IsNumber(r) {
			token.WriteRune(r)
		} else {
			flush()
		}
	}
	flush()
	return tokens
}
