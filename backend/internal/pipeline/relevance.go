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

// Tokens are stemmed before stop words are removed, so the stop word list
// must also hold the stemmed forms ("does" -> "doe", "having" -> "hav").
func init() {
	for word := range relevanceStopWords {
		relevanceStopWords[stem(word)] = struct{}{}
	}
}

// queryProfile caches the tokenized form of a query so that scoring many
// candidates does not re-tokenize the query for every comparison.
type queryProfile struct {
	terms  map[string]struct{}
	phrase string
	// weights, when set, holds each term's importance from the candidate pool
	// (see withPoolWeights). Nil: every term counts the same.
	weights map[string]float64
}

const (
	// poolWeightFloor is the least a query term weighs, however few candidates
	// use it: a word no engine returned still counts for something.
	poolWeightFloor = 0.5
	// weightedCoverageMin is the share of the query's total term weight a
	// result must cover to be relevant through pool weighting.
	weightedCoverageMin = 0.6
)

// withPoolWeights returns a copy of the profile whose terms are weighted by
// how many candidate results (titles and snippets, across all engines) use
// them: weight = floor + (1 - floor) x share of candidates containing the term.
//
// A natural-language question carries words the answer pages need not repeat
// ("who *wrote* the hobbit": pages about The Hobbit rarely say "wrote"). The
// engines' results show which words the question is about: nearly all of them
// mention "hobbit", few mention "wrote". With weights, a result covering the
// heavily used terms is relevant even when it misses an incidental one, while
// a decoy page that matches only a word the other engines' results do not use
// (a dictionary entry for "meaning") still is not.
func (q queryProfile) withPoolWeights(texts []string) queryProfile {
	sets := make([][]map[string]struct{}, len(texts))
	for i, text := range texts {
		sets[i] = []map[string]struct{}{tokenSet(text)}
	}
	return q.withPoolWeightsTokens(sets)
}

// withPoolWeightsTokens is withPoolWeights for candidates already tokenized:
// each candidate is a list of token sets (title, snippet), and a term counts
// once per candidate when any of its sets holds it.
func (q queryProfile) withPoolWeightsTokens(candidates [][]map[string]struct{}) queryProfile {
	if len(q.terms) < 2 || len(candidates) == 0 {
		return q
	}
	counts := make(map[string]int, len(q.terms))
	for _, sets := range candidates {
		for term := range q.terms {
			for _, tokens := range sets {
				if _, ok := tokens[term]; ok {
					counts[term]++
					break
				}
			}
		}
	}
	weights := make(map[string]float64, len(q.terms))
	for term := range q.terms {
		share := float64(counts[term]) / float64(len(candidates))
		weights[term] = poolWeightFloor + (1-poolWeightFloor)*share
	}
	q.weights = weights
	return q
}

func newQueryProfile(query string) queryProfile {
	query = stripSearchOperators(query)
	return queryProfile{terms: meaningfulTokens(query), phrase: normalizedPhrase(query)}
}

// stripSearchOperators removes query syntax that must not be treated as
// required words: negated terms (-car, -"used cars", -site:x), field operators
// (site:, filetype:, inurl:), Boolean keywords, and quotes. Field operators that
// carry content words (intitle:foo) keep the word.
func stripSearchOperators(query string) string {
	var kept []string
	inNegatedPhrase := false
	for _, field := range strings.Fields(query) {
		if inNegatedPhrase {
			if strings.HasSuffix(field, `"`) {
				inNegatedPhrase = false
			}
			continue
		}
		if strings.HasPrefix(field, "-") && len(field) > 1 {
			if strings.HasPrefix(field, `-"`) && !strings.HasSuffix(field[2:], `"`) {
				inNegatedPhrase = true
			}
			continue
		}
		if field == "OR" || field == "AND" {
			continue
		}
		lower := strings.ToLower(field)
		skip := false
		for _, op := range []string{"site:", "filetype:", "ext:", "inurl:", "allinurl:"} {
			if strings.HasPrefix(lower, op) {
				skip = true
				break
			}
		}
		if skip {
			continue
		}
		for _, op := range []string{"intitle:", "allintitle:", "intext:", "allintext:", "inbody:"} {
			if strings.HasPrefix(lower, op) {
				field = field[len(op):]
				break
			}
		}
		field = strings.Trim(field, `"()`)
		if field != "" {
			kept = append(kept, field)
		}
	}
	if len(kept) == 0 {
		return query // nothing but operators; fall back to the literal text
	}
	return strings.Join(kept, " ")
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
	score, relevant, _ = q.match(title, snippet)
	return score, relevant
}

// match scores a result and also reports coverage: the fraction of query
// terms found anywhere in the title or snippet.
func (q queryProfile) match(title, snippet string) (score float64, relevant bool, coverage float64) {
	return q.matchTokens(title, snippet, tokenSet(title), tokenSet(snippet))
}

// matchTokens is match with the title and snippet already tokenized, so the
// pipeline can tokenize each candidate once for both pool weights and scoring.
func (q queryProfile) matchTokens(title, snippet string, titleTokens, snippetTokens map[string]struct{}) (score float64, relevant bool, coverage float64) {
	queryTerms := q.terms
	if len(queryTerms) == 0 {
		return 1, true, 1
	}

	matched := 0
	titleMatched := 0
	snippetMatched := 0
	weightMatched, weightTotal := 0.0, 0.0
	missingCompoundIdentifier := false
	for term := range queryTerms {
		_, inTitle := titleTokens[term]
		_, inSnippet := snippetTokens[term]
		weightTotal += q.weights[term]
		if inSnippet {
			snippetMatched++
		}
		if inTitle || inSnippet {
			matched++
			weightMatched += q.weights[term]
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
	coverage = float64(matched) / n
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
	// With pool weights, covering most of the query's weight also counts: the
	// words the engines' results are about, not every word of the question.
	weighted := weightTotal > 0 && weightMatched/weightTotal >= weightedCoverageMin
	return score, !missingCompoundIdentifier && (phraseMatch || matched >= minimumCoverage || weighted), coverage
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
	for _, token := range Tokens(value) {
		tokens[token] = struct{}{}
	}
	return tokens
}

// Tokens splits text into normalized, stemmed tokens in reading order,
// repeats included. It applies the same rules as result relevance scoring:
// lower-casing, C++/C#/F#/.NET kept as identifiers, golang -> go, light
// English stemming, and one token per CJK character.
func Tokens(value string) []string {
	var tokens []string
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
		tokens = append(tokens, stem(normalized))
		token.Reset()
	}
	for _, r := range value {
		switch {
		case isCJK(r):
			// CJK scripts do not separate words with spaces; index each
			// character so queries can match inside longer runs of text.
			flush()
			tokens = append(tokens, string(r))
		case unicode.IsLetter(r) || unicode.IsNumber(r):
			token.WriteRune(r)
		default:
			flush()
		}
	}
	flush()
	return tokens
}

// QueryTerms returns the distinct tokens of a query that should count when
// matching text: search operators (site:, -term, quotes) and stop words are
// removed, unless the query consists of nothing else.
func QueryTerms(query string) []string {
	set := meaningfulTokens(stripSearchOperators(query))
	terms := make([]string, 0, len(set))
	for _, token := range Tokens(stripSearchOperators(query)) {
		if _, ok := set[token]; ok {
			terms = append(terms, token)
			delete(set, token)
		}
	}
	return terms
}

// NormalizedPhrase lower-cases text and reduces it to words separated by
// single spaces, for exact-phrase comparisons.
func NormalizedPhrase(value string) string {
	return normalizedPhrase(value)
}

func isCJK(r rune) bool {
	return unicode.In(r, unicode.Han, unicode.Hiragana, unicode.Katakana, unicode.Hangul)
}

// stemExceptions are words whose trailing "s" is not a plural marker.
var stemExceptions = map[string]struct{}{
	"news": {}, "series": {}, "species": {}, "always": {}, "across": {}, "perhaps": {}, "towards": {}, "physics": {}, "mathematics": {},
}

// stem reduces common English inflections so "laptops" matches "laptop" and
// "caching" matches "cache". It is deliberately light: it only needs to map
// the query and the result text to the same form, not produce real words.
func stem(token string) string {
	if len(token) <= 3 {
		return token
	}
	for i := 0; i < len(token); i++ {
		if c := token[i]; c < 'a' || c > 'z' {
			return token // digits, non-ASCII scripts, identifiers
		}
	}
	if _, keep := stemExceptions[token]; keep {
		return token
	}
	switch {
	case strings.HasSuffix(token, "ies") && len(token) > 4:
		token = token[:len(token)-3] + "y"
	case strings.HasSuffix(token, "sses"), strings.HasSuffix(token, "xes"), strings.HasSuffix(token, "ches"), strings.HasSuffix(token, "shes"):
		token = token[:len(token)-2]
	case strings.HasSuffix(token, "s") && !strings.HasSuffix(token, "ss") && !strings.HasSuffix(token, "us") && !strings.HasSuffix(token, "is"):
		token = token[:len(token)-1]
	}
	switch {
	case strings.HasSuffix(token, "ing") && len(token) > 5:
		token = undouble(token[:len(token)-3])
	case strings.HasSuffix(token, "ed") && len(token) > 4:
		token = undouble(token[:len(token)-2])
	}
	if len(token) > 3 && strings.HasSuffix(token, "e") {
		token = token[:len(token)-1]
	}
	return token
}

// undouble collapses a doubled final consonant left by suffix removal
// ("runn" -> "run"), except for letters that are commonly doubled in stems.
func undouble(token string) string {
	n := len(token)
	if n < 3 || token[n-1] != token[n-2] {
		return token
	}
	switch token[n-1] {
	case 'l', 's', 'z', 'a', 'e', 'i', 'o', 'u':
		return token
	}
	return token[:n-1]
}
