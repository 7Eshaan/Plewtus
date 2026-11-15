package extractor

import (
	"math"
	"net/url"
	"regexp"
	"sort"
	"strings"
	"unicode/utf8"

	"golang.org/x/net/html"
	"golang.org/x/net/html/atom"

	"plewtus/internal/pipeline"
)

// Passage selection: when a page is longer than the caller's character
// budget, return the parts that answer the query instead of only its start.
//
// The page is split into blocks (paragraphs, lists, tables, code), grouped
// into chunks of roughly chunkMinChars-chunkMaxChars under their headings,
// and scored against the query with BM25 over the page's own chunks. The
// intro is always kept; the best chunks fill the rest of the budget, then
// their neighbours. Chosen chunks are emitted in page order under their
// headings, with a gap marker wherever text was left out.

const (
	// maxConvertChars caps how much of a page is converted and scored
	// (long PDFs such as specifications run to several hundred thousand).
	maxConvertChars = 500_000
	chunkMinChars   = 300
	chunkMaxChars   = 800
	// GapMarker shows where page text was left out between passages.
	GapMarker = "[…]"

	bm25K1 = 1.2
	bm25B  = 0.75
)

// block is one paragraph-level piece of the page, rendered both ways.
type block struct {
	heading    int // 1-6 for a heading block, else 0
	md, text   string
	splittable bool // plain prose that may be split at sentence boundaries
}

// chunk is a run of blocks under one heading path.
type chunk struct {
	path     []string // heading texts from the outermost level inward
	levels   []int    // heading level of each path entry
	md, text string
}

var leafBlockTags = map[atom.Atom]bool{
	atom.P: true, atom.Pre: true, atom.Table: true, atom.Ul: true, atom.Ol: true, atom.Dl: true,
	atom.Blockquote: true, atom.Figure: true, atom.Hr: true,
	atom.H1: true, atom.H2: true, atom.H3: true, atom.H4: true, atom.H5: true, atom.H6: true,
}

// splitListTags are lists that may be split into items when too long.
var splitListTags = map[atom.Atom]bool{atom.Ul: true, atom.Ol: true, atom.Dl: true}

// renderedLength is the Markdown length of a node.
func renderedLength(n *html.Node, base *url.URL) int {
	var buf strings.Builder
	renderMarkdownNode(n, &buf, &markdownState{base: base})
	return utf8.RuneCountInString(cleanMarkdownOutput(buf.String()))
}

var containerTags = map[atom.Atom]bool{
	atom.Div: true, atom.Section: true, atom.Article: true, atom.Main: true, atom.Header: true,
	atom.Body: true, atom.Form: true, atom.Details: true, atom.Center: true,
}

var headingLevels = map[atom.Atom]int{
	atom.H1: 1, atom.H2: 2, atom.H3: 3, atom.H4: 4, atom.H5: 5, atom.H6: 6,
}

// collectBlocks flattens a content node into paragraph-level blocks. Runs of
// inline content directly inside containers become their own block.
func collectBlocks(root *html.Node, base *url.URL) []block {
	var blocks []block
	total := 0
	var inline []*html.Node

	emit := func(nodes []*html.Node, heading int, splittable bool) {
		if total >= maxConvertChars {
			return
		}
		var mdBuf, textBuf strings.Builder
		state := &markdownState{base: base}
		for _, n := range nodes {
			if heading > 0 {
				mdBuf.WriteString(renderInline(n, state))
			} else {
				renderMarkdownNode(n, &mdBuf, state)
			}
			renderTextNode(n, &textBuf)
		}
		md := cleanMarkdownOutput(mdBuf.String())
		text := cleanTextOutput(textBuf.String())
		if strings.TrimSpace(text) == "" && !strings.HasPrefix(md, "![") {
			return
		}
		total += utf8.RuneCountInString(md)
		blocks = append(blocks, block{heading: heading, md: md, text: text, splittable: splittable})
	}
	// emitItem renders one list item as a single-item list, keeping its
	// number in an ordered list.
	emitItem := func(item *html.Node, ordinal int) {
		var mdBuf, textBuf strings.Builder
		state := &markdownState{base: base, inList: true, isOrdered: ordinal > 0, listIndex: ordinal}
		renderMarkdownNode(item, &mdBuf, state)
		renderTextNode(item, &textBuf)
		md, text := cleanMarkdownOutput(mdBuf.String()), cleanTextOutput(textBuf.String())
		if text == "" || total >= maxConvertChars {
			return
		}
		total += utf8.RuneCountInString(md)
		blocks = append(blocks, block{md: md, text: text})
	}
	flushInline := func() {
		if len(inline) > 0 {
			emit(inline, 0, true)
			inline = nil
		}
	}

	var walk func(n *html.Node)
	walk = func(n *html.Node) {
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			switch {
			case c.Type == html.ElementNode && leafBlockTags[c.DataAtom]:
				flushInline()
				if c.DataAtom == atom.Hr {
					continue
				}
				// Lists and definition lists longer than a chunk are split
				// into their items: API references put whole functions in one
				// <dl>, which would otherwise be too big to ever select. Code
				// and tables stay whole.
				if splitListTags[c.DataAtom] && renderedLength(c, base) > chunkMaxChars {
					ordinal := 0
					for item := c.FirstChild; item != nil; item = item.NextSibling {
						if item.Type != html.ElementNode {
							continue
						}
						switch item.DataAtom {
						case atom.Li:
							ordinal++
							listOrdinal := 0
							if c.DataAtom == atom.Ol {
								listOrdinal = ordinal
							}
							emitItem(item, listOrdinal)
						case atom.Dt:
							emit([]*html.Node{item}, 0, false)
						default: // <dd> and wrappers: their own blocks
							walk(item)
							flushInline()
						}
					}
					continue
				}
				emit([]*html.Node{c}, headingLevels[c.DataAtom], c.DataAtom == atom.P || c.DataAtom == atom.Blockquote)
			case c.Type == html.ElementNode && (containerTags[c.DataAtom] || hasBlockDescendant(c)):
				flushInline()
				walk(c)
			default:
				inline = append(inline, c)
			}
		}
	}
	walk(root)
	flushInline()
	return blocks
}

func hasBlockDescendant(n *html.Node) bool {
	for c := n.FirstChild; c != nil; c = c.NextSibling {
		if c.Type == html.ElementNode && (leafBlockTags[c.DataAtom] || containerTags[c.DataAtom] || hasBlockDescendant(c)) {
			return true
		}
	}
	return false
}

// buildChunks groups blocks into chunks under their heading path. Long prose
// blocks are split at sentence boundaries; code, tables, and lists never are.
func buildChunks(blocks []block, maxChars int) []chunk {
	if maxChars <= 0 {
		maxChars = chunkMaxChars
	}
	minChars := maxChars * chunkMinChars / chunkMaxChars
	var chunks []chunk
	var path []string
	var levels []int
	var md, text []string
	size := 0

	closeChunk := func() {
		if len(md) == 0 {
			return
		}
		chunks = append(chunks, chunk{
			path:   append([]string(nil), path...),
			levels: append([]int(nil), levels...),
			md:     strings.Join(md, "\n\n"),
			text:   strings.Join(text, "\n\n"),
		})
		md, text, size = nil, nil, 0
	}
	add := func(m, t string) {
		n := utf8.RuneCountInString(m)
		if size >= minChars && size+n > maxChars {
			closeChunk()
		}
		md = append(md, m)
		text = append(text, t)
		size += n
	}

	for _, b := range blocks {
		if b.heading > 0 {
			closeChunk()
			for len(levels) > 0 && levels[len(levels)-1] >= b.heading {
				levels = levels[:len(levels)-1]
				path = path[:len(path)-1]
			}
			path = append(path, b.md)
			levels = append(levels, b.heading)
			continue
		}
		if b.splittable && utf8.RuneCountInString(b.md) > maxChars {
			for _, piece := range splitProse(b.md, maxChars) {
				add(piece, markdownToText(piece))
			}
			continue
		}
		add(b.md, b.text)
	}
	closeChunk()
	return chunks
}

// splitProse cuts a long paragraph into pieces of at most chunkMaxChars at
// sentence boundaries (or anywhere, for a single overlong sentence).
func splitProse(md string, maxChars int) []string {
	var pieces []string
	var cur strings.Builder
	for _, sentence := range splitSentences(md) {
		if cur.Len() > 0 && utf8.RuneCountInString(cur.String())+1+utf8.RuneCountInString(sentence) > maxChars {
			pieces = append(pieces, cur.String())
			cur.Reset()
		}
		if cur.Len() > 0 {
			cur.WriteByte(' ')
		}
		cur.WriteString(sentence)
	}
	if cur.Len() > 0 {
		pieces = append(pieces, cur.String())
	}
	return pieces
}

var (
	mdLink   = regexp.MustCompile(`!?\[([^\]]*)\]\([^)]*\)`)
	mdMarker = regexp.MustCompile("(\\*\\*|__|`)")
)

// markdownToText strips inline Markdown syntax from a prose piece.
func markdownToText(md string) string {
	return mdMarker.ReplaceAllString(mdLink.ReplaceAllString(md, "$1"), "")
}

// linkShare is the fraction of a chunk's Markdown taken up by links.
func linkShare(md string) float64 {
	if md == "" {
		return 0
	}
	linked := 0
	for _, m := range mdLink.FindAllString(md, -1) {
		linked += len(m)
	}
	return float64(linked) / float64(len(md))
}

type scoredChunk struct {
	index  int
	score  float64
	tokens map[string]bool
}

// scoreChunks ranks chunks against the query with BM25 (k1=1.2, b=0.75)
// computed over the page's own chunks, plus heading, phrase, and coverage
// signals. It returns nil when the query has no usable terms.
func scoreChunks(chunks []chunk, query string) []scoredChunk {
	terms := pipeline.QueryTerms(query)
	if len(terms) == 0 || len(chunks) == 0 {
		return nil
	}
	phrase := ""
	if len(terms) > 1 {
		phrase = pipeline.NormalizedPhrase(query)
	}

	tfs := make([]map[string]int, len(chunks))
	lengths := make([]int, len(chunks))
	df := make(map[string]int)
	totalLen := 0
	for i, c := range chunks {
		tokens := pipeline.Tokens(c.text)
		tf := make(map[string]int, len(tokens))
		for _, t := range tokens {
			tf[t]++
		}
		for _, term := range terms {
			if tf[term] > 0 {
				df[term]++
			}
		}
		tfs[i] = tf
		lengths[i] = len(tokens)
		totalLen += len(tokens)
	}
	avgLen := math.Max(1, float64(totalLen)/float64(len(chunks)))
	n := float64(len(chunks))

	scored := make([]scoredChunk, len(chunks))
	for i, c := range chunks {
		heading := make(map[string]bool)
		for _, t := range pipeline.Tokens(strings.Join(c.path, " ")) {
			heading[t] = true
		}
		score, matched := 0.0, 0
		for _, term := range terms {
			idf := math.Log(1 + (n-float64(df[term])+0.5)/(float64(df[term])+0.5))
			if tf := float64(tfs[i][term]); tf > 0 {
				matched++
				score += idf * tf * (bm25K1 + 1) / (tf + bm25K1*(1-bm25B+bm25B*float64(lengths[i])/avgLen))
			}
			if heading[term] {
				score += 0.5 * idf
			}
		}
		// Chunks matching more of the query's distinct terms rank higher.
		score *= 0.5 + 0.5*float64(matched)/float64(len(terms))
		if phrase != "" && strings.Contains(" "+pipeline.NormalizedPhrase(c.text)+" ", " "+phrase+" ") {
			score *= 1.5
		}
		if linkShare(c.md) > 0.5 {
			score *= 0.3
		}
		if utf8.RuneCountInString(c.text) < 200 && isBoilerplateSentence(c.text) {
			score *= 0.2
		}
		tokenSet := make(map[string]bool, len(tfs[i]))
		for t := range tfs[i] {
			tokenSet[t] = true
		}
		scored[i] = scoredChunk{index: i, score: score, tokens: tokenSet}
	}
	return scored
}

// nearDuplicate reports whether two chunks share most of their vocabulary.
func nearDuplicate(a, b map[string]bool) bool {
	if len(a) == 0 || len(b) == 0 {
		return false
	}
	shared := 0
	for t := range a {
		if b[t] {
			shared++
		}
	}
	union := len(a) + len(b) - shared
	return float64(shared)/float64(union) > 0.8
}

// assemblePassages renders the chosen chunks in page order. Headings are
// written when the heading path changes; GapMarker marks omitted text.
func assemblePassages(chunks []chunk, chosen []int, markdown bool) string {
	sort.Ints(chosen)
	var b strings.Builder
	var lastPath []string
	prev := -1
	for _, idx := range chosen {
		c := chunks[idx]
		if b.Len() > 0 {
			b.WriteString("\n\n")
		}
		if idx > prev+1 {
			b.WriteString(GapMarker + "\n\n")
		}
		// Write the headings from the first level where the path changed.
		common := 0
		for common < len(c.path) && common < len(lastPath) && c.path[common] == lastPath[common] {
			common++
		}
		if markdown {
			for i := common; i < len(c.path); i++ {
				level := c.levels[i]
				if level > 4 {
					level = 4
				}
				b.WriteString(strings.Repeat("#", level) + " " + c.path[i] + "\n\n")
			}
			b.WriteString(c.md)
		} else {
			if common < len(c.path) {
				b.WriteString(strings.Join(c.path, " › ") + "\n\n")
			}
			b.WriteString(c.text)
		}
		lastPath = c.path
		prev = idx
	}
	if prev >= 0 && prev < len(chunks)-1 {
		b.WriteString("\n\n" + GapMarker)
	}
	return b.String()
}

// selectPassages returns the parts of a page most relevant to query that fit
// in budget characters, or "" when no passage selection applies (no query
// terms, too few chunks, or nothing on the page matches the query).
func selectPassages(root *html.Node, base *url.URL, query string, budget int, markdown bool) string {
	if budget < chunkMinChars || strings.TrimSpace(query) == "" {
		return ""
	}
	// Chunks scale with the budget so several passages can fit even in a
	// small one: a third of the budget, between 200 and chunkMaxChars.
	chunkSize := budget / 3
	if chunkSize > chunkMaxChars {
		chunkSize = chunkMaxChars
	} else if chunkSize < 200 {
		chunkSize = 200
	}
	chunks := buildChunks(collectBlocks(root, base), chunkSize)
	if len(chunks) < 2 {
		return ""
	}
	scored := scoreChunks(chunks, query)
	if scored == nil {
		return ""
	}

	byScore := append([]scoredChunk(nil), scored...)
	sort.SliceStable(byScore, func(i, j int) bool { return byScore[i].score > byScore[j].score })
	if byScore[0].score <= 0 {
		return "" // the query does not occur on the page; the lead is the best guess
	}

	var chosen []int
	picked := make(map[int]bool)
	fits := func(idx int) bool {
		out := assemblePassages(chunks, append(append([]int(nil), chosen...), idx), markdown)
		return utf8.RuneCountInString(out) <= budget
	}
	tryAdd := func(idx int) {
		if idx < 0 || idx >= len(chunks) || picked[idx] || !fits(idx) {
			return
		}
		for _, other := range chosen {
			if nearDuplicate(scored[idx].tokens, scored[other].tokens) {
				return
			}
		}
		chosen = append(chosen, idx)
		picked[idx] = true
	}

	// 1. The best match, then the intro (it usually summarizes the page;
	// with a small budget the match comes first).
	var relevant []int
	tryAdd(byScore[0].index)
	if picked[byScore[0].index] {
		relevant = append(relevant, byScore[0].index)
	}
	tryAdd(0)
	// 2. The other chunks that match the query, best first.
	for _, sc := range byScore[1:] {
		if sc.score <= 0 {
			break
		}
		before := len(chosen)
		tryAdd(sc.index)
		if len(chosen) > before {
			relevant = append(relevant, sc.index)
		}
	}
	if len(relevant) == 0 {
		return ""
	}
	// 3. Context: the text right after, then right before, each match.
	for _, idx := range relevant {
		tryAdd(idx + 1)
		tryAdd(idx - 1)
	}
	// 4. Continue from the top of the page while the text still fits, so the
	// lead reads contiguously instead of picking scattered small pieces.
	for idx := range chunks {
		if !picked[idx] && !fits(idx) {
			break
		}
		tryAdd(idx)
	}
	return assemblePassages(chunks, chosen, markdown)
}
