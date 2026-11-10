package extractor

import (
	"fmt"
	"net/url"
	"sort"
	"strings"
	"unicode"

	"golang.org/x/net/html"
	"golang.org/x/net/html/atom"
)

// ToMarkdownWithBase converts HTML to Markdown, resolving relative link and
// image URLs against base so they remain usable outside the source page.
func ToMarkdownWithBase(n *html.Node, maxChars int, base *url.URL) string {
	var buf strings.Builder
	renderMarkdownNode(n, &buf, &markdownState{base: base})

	raw := buf.String()
	cleaned := cleanMarkdownOutput(raw)

	if maxChars > 0 && len(cleaned) > maxChars {
		return closeOpenFence(truncateCleanly(cleaned, maxChars))
	}

	return cleaned
}

// closeOpenFence terminates a code block that truncation cut in half, so the
// rest of whatever the Markdown is embedded in is not swallowed as code.
func closeOpenFence(markdown string) string {
	open := false
	for _, line := range strings.Split(markdown, "\n") {
		if strings.HasPrefix(strings.TrimSpace(line), "```") {
			open = !open
		}
	}
	if open {
		return strings.TrimRight(markdown, "\n") + "\n```"
	}
	return markdown
}

// ToText converts an HTML node and its subtree into clean, normalized plain text.
func ToText(n *html.Node, maxChars int) string {
	var buf strings.Builder
	renderTextNode(n, &buf)

	raw := buf.String()
	cleaned := cleanTextOutput(raw)

	if maxChars > 0 && len(cleaned) > maxChars {
		return truncateCleanly(cleaned, maxChars)
	}

	return cleaned
}

// ExtractHighlights selects 3 to 5 key, high-density highlight sentences from the content.
func ExtractHighlights(text string, query string, maxHighlights int) []string {
	if maxHighlights <= 0 {
		maxHighlights = 4
	}

	sentences := splitSentences(text)
	if len(sentences) == 0 {
		return nil
	}

	queryTerms := strings.Fields(strings.ToLower(query))

	type scoredSentence struct {
		sentence string
		score    float64
		index    int
	}

	var candidates []scoredSentence
	seen := make(map[string]bool)

	for idx, s := range sentences {
		s = strings.TrimSpace(s)
		norm := strings.ToLower(s)

		// Filter out very short or repetitive sentences
		if len(s) < 25 || seen[norm] {
			continue
		}
		seen[norm] = true

		// Skip boilerplate phrases
		if isBoilerplateSentence(norm) {
			continue
		}

		score := 0.0

		// Boost sentences matching query terms
		for _, term := range queryTerms {
			if len(term) > 2 && strings.Contains(norm, term) {
				score += 2.0
			}
		}

		// Length preference: 50 - 200 chars is ideal for an AI highlight
		if len(s) >= 40 && len(s) <= 220 {
			score += 1.5
		} else if len(s) > 350 {
			score -= 1.0 // penalize run-on sentences
		}

		// Early sentences in an article are often the most informative summary
		if idx < 5 {
			score += 1.0 - (float64(idx) * 0.15)
		}

		candidates = append(candidates, scoredSentence{
			sentence: s,
			score:    score,
			index:    idx,
		})
	}

	if len(candidates) == 0 {
		// Fallback to first non-empty sentences if none scored well
		var fallback []string
		for _, s := range sentences {
			s = strings.TrimSpace(s)
			if len(s) >= 20 {
				fallback = append(fallback, s)
				if len(fallback) >= maxHighlights {
					break
				}
			}
		}
		return fallback
	}

	// Sort by score descending in O(n log n) for long articles.
	sort.SliceStable(candidates, func(i, j int) bool {
		return candidates[i].score > candidates[j].score
	})

	// Pick top N
	count := maxHighlights
	if len(candidates) < count {
		count = len(candidates)
	}

	top := candidates[:count]

	// Re-sort selected sentences by original document order for natural reading flow
	sort.Slice(top, func(i, j int) bool {
		return top[i].index < top[j].index
	})

	result := make([]string, len(top))
	for i, item := range top {
		result[i] = item.sentence
	}

	return result
}

type markdownState struct {
	inList      bool
	listIndex   int
	isOrdered   bool
	inCodeBlock bool
	base        *url.URL
}

// resolveURL makes href absolute against the page base. It returns "" for
// links that are useless outside the page (javascript:, in-page fragments).
func (s *markdownState) resolveURL(href string) string {
	href = strings.TrimSpace(href)
	lower := strings.ToLower(href)
	if href == "" || strings.HasPrefix(href, "#") || strings.HasPrefix(lower, "javascript:") || strings.HasPrefix(lower, "data:") {
		return ""
	}
	if s.base == nil {
		return href
	}
	ref, err := url.Parse(href)
	if err != nil {
		return href
	}
	return s.base.ResolveReference(ref).String()
}

// renderInline renders a node's children to a single trimmed line.
func renderInline(n *html.Node, state *markdownState) string {
	var b strings.Builder
	renderChildren(n, &b, state)
	return strings.TrimSpace(normalizeInlineWhitespace(b.String()))
}

func renderMarkdownNode(n *html.Node, buf *strings.Builder, state *markdownState) {
	if n == nil {
		return
	}

	switch n.Type {
	case html.TextNode:
		text := n.Data
		if !state.inCodeBlock {
			text = normalizeInlineWhitespace(text)
		}
		buf.WriteString(text)
		return

	case html.ElementNode:
		switch n.DataAtom {
		case atom.H1, atom.H2, atom.H3, atom.H4, atom.H5, atom.H6:
			level := map[atom.Atom]int{atom.H1: 1, atom.H2: 2, atom.H3: 3}[n.DataAtom]
			if level == 0 {
				level = 4
			}
			if text := renderInline(n, state); text != "" {
				buf.WriteString("\n\n" + strings.Repeat("#", level) + " " + text + "\n\n")
			}
			return
		case atom.P:
			buf.WriteString("\n\n")
			renderChildren(n, buf, state)
			buf.WriteString("\n\n")
			return
		case atom.Br:
			buf.WriteString("\n")
			return
		case atom.Hr:
			buf.WriteString("\n\n---\n\n")
			return
		case atom.Blockquote:
			buf.WriteString("\n\n> ")
			renderChildren(n, buf, state)
			buf.WriteString("\n\n")
			return
		case atom.Ul:
			oldInList, oldOrdered := state.inList, state.isOrdered
			state.inList = true
			state.isOrdered = false
			buf.WriteString("\n")
			renderChildren(n, buf, state)
			buf.WriteString("\n")
			state.inList, state.isOrdered = oldInList, oldOrdered
			return
		case atom.Ol:
			oldInList, oldOrdered, oldIdx := state.inList, state.isOrdered, state.listIndex
			state.inList = true
			state.isOrdered = true
			state.listIndex = 1
			buf.WriteString("\n")
			renderChildren(n, buf, state)
			buf.WriteString("\n")
			state.inList, state.isOrdered, state.listIndex = oldInList, oldOrdered, oldIdx
			return
		case atom.Li:
			var itemBuf strings.Builder
			renderChildren(n, &itemBuf, state)
			itemText := strings.TrimSpace(itemBuf.String())
			if !state.inCodeBlock {
				var lines []string
				for _, line := range strings.Split(itemText, "\n") {
					if line = strings.TrimSpace(line); line != "" {
						lines = append(lines, line)
					}
				}
				itemText = strings.Join(lines, "\n")
			}
			if itemText == "" {
				return
			}
			if state.isOrdered {
				buf.WriteString(fmt.Sprintf("%d. %s\n", state.listIndex, itemText))
				state.listIndex++
			} else {
				buf.WriteString(fmt.Sprintf("* %s\n", itemText))
			}
			return
		case atom.Img:
			alt := strings.TrimSpace(getAttr(n, "alt"))
			src := state.resolveURL(getAttr(n, "src"))
			if alt != "" && src != "" {
				buf.WriteString(fmt.Sprintf("![%s](%s)", alt, src))
			}
			return
		case atom.Pre:
			oldInCode := state.inCodeBlock
			state.inCodeBlock = true
			buf.WriteString("\n\n```\n")
			renderChildren(n, buf, state)
			buf.WriteString("\n```\n\n")
			state.inCodeBlock = oldInCode
			return
		case atom.Code:
			if state.inCodeBlock {
				renderChildren(n, buf, state)
			} else {
				buf.WriteString("`")
				renderChildren(n, buf, state)
				buf.WriteString("`")
			}
			return
		case atom.Strong, atom.B, atom.Em, atom.I:
			if state.inCodeBlock {
				renderChildren(n, buf, state)
				return
			}
			marker := "**"
			if n.DataAtom == atom.Em || n.DataAtom == atom.I {
				marker = "*"
			}
			if text := renderInline(n, state); text != "" {
				buf.WriteString(marker + text + marker)
			}
			return
		case atom.A:
			text := renderInline(n, state)
			if text == "" {
				return
			}
			if href := state.resolveURL(getAttr(n, "href")); href != "" && !state.inCodeBlock {
				buf.WriteString(fmt.Sprintf("[%s](%s)", text, href))
			} else {
				buf.WriteString(text)
			}
			return
		case atom.Table:
			buf.WriteString("\n\n")
			renderTable(n, buf)
			buf.WriteString("\n\n")
			return
		}
	}

	renderChildren(n, buf, state)
}

func renderChildren(n *html.Node, buf *strings.Builder, state *markdownState) {
	for c := n.FirstChild; c != nil; c = c.NextSibling {
		renderMarkdownNode(c, buf, state)
	}
}

func renderTable(table *html.Node, buf *strings.Builder) {
	var rows [][]string
	var walk func(*html.Node)
	walk = func(n *html.Node) {
		if n.Type == html.ElementNode && n.DataAtom == atom.Tr {
			var cols []string
			for cell := n.FirstChild; cell != nil; cell = cell.NextSibling {
				if cell.Type == html.ElementNode && (cell.DataAtom == atom.Td || cell.DataAtom == atom.Th) {
					cols = append(cols, extractNodeText(cell))
				}
			}
			if len(cols) > 0 {
				rows = append(rows, cols)
			}
		}
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			walk(c)
		}
	}
	walk(table)

	if len(rows) == 0 {
		return
	}

	maxCols := 0
	for _, r := range rows {
		if len(r) > maxCols {
			maxCols = len(r)
		}
	}

	for i, r := range rows {
		buf.WriteString("|")
		for c := 0; c < maxCols; c++ {
			val := ""
			if c < len(r) {
				val = r[c]
			}
			buf.WriteString(" " + strings.ReplaceAll(val, "|", "\\|") + " |")
		}
		buf.WriteString("\n")

		if i == 0 {
			buf.WriteString("|")
			for c := 0; c < maxCols; c++ {
				buf.WriteString(" --- |")
			}
			buf.WriteString("\n")
		}
	}
}

func renderTextNode(n *html.Node, buf *strings.Builder) {
	if n == nil {
		return
	}

	if n.Type == html.TextNode {
		text := strings.TrimSpace(normalizeInlineWhitespace(n.Data))
		if text != "" {
			buf.WriteString(text)
			buf.WriteString(" ")
		}
		return
	}

	if n.Type == html.ElementNode {
		isBlock := isBlockElement(n.DataAtom)
		if isBlock {
			buf.WriteString("\n\n")
		}

		for c := n.FirstChild; c != nil; c = c.NextSibling {
			renderTextNode(c, buf)
		}

		if isBlock {
			buf.WriteString("\n\n")
		}
		return
	}

	for c := n.FirstChild; c != nil; c = c.NextSibling {
		renderTextNode(c, buf)
	}
}

func isBlockElement(a atom.Atom) bool {
	switch a {
	case atom.P, atom.H1, atom.H2, atom.H3, atom.H4, atom.H5, atom.H6,
		atom.Ul, atom.Ol, atom.Li, atom.Blockquote, atom.Pre, atom.Table, atom.Tr:
		return true
	default:
		return false
	}
}

// normalizeInlineWhitespace collapses runs of whitespace to a single space
// while keeping one leading/trailing space, so text adjacent to inline
// elements ("See <a>docs</a>") does not get glued together.
func normalizeInlineWhitespace(s string) string {
	if s == "" {
		return ""
	}
	inner := strings.Join(strings.Fields(s), " ")
	if inner == "" {
		return " "
	}
	runes := []rune(s)
	if unicode.IsSpace(runes[0]) {
		inner = " " + inner
	}
	if unicode.IsSpace(runes[len(runes)-1]) {
		inner += " "
	}
	return inner
}

func cleanMarkdownOutput(s string) string {
	lines := strings.Split(s, "\n")
	var cleaned []string
	consecutiveBlank := 0
	inFence := false

	for _, line := range lines {
		trimmed := strings.TrimRightFunc(line, unicode.IsSpace)
		if strings.HasPrefix(strings.TrimSpace(trimmed), "```") {
			inFence = !inFence
			trimmed = strings.TrimSpace(trimmed)
		} else if !inFence {
			// Leading spaces outside code blocks are artifacts of inline
			// whitespace preservation; Markdown would read them as indentation.
			trimmed = strings.TrimLeftFunc(trimmed, unicode.IsSpace)
		}
		if trimmed == "" {
			consecutiveBlank++
			if consecutiveBlank <= 2 {
				cleaned = append(cleaned, "")
			}
		} else {
			consecutiveBlank = 0
			cleaned = append(cleaned, trimmed)
		}
	}

	return strings.TrimSpace(strings.Join(cleaned, "\n"))
}

func cleanTextOutput(s string) string {
	paragraphs := strings.Split(s, "\n\n")
	var cleaned []string
	for _, p := range paragraphs {
		fields := strings.Fields(p)
		if len(fields) > 0 {
			cleaned = append(cleaned, strings.Join(fields, " "))
		}
	}
	return strings.TrimSpace(strings.Join(cleaned, "\n\n"))
}

func splitSentences(text string) []string {
	var sentences []string
	var current strings.Builder

	runes := []rune(text)
	length := len(runes)

	for i := 0; i < length; i++ {
		r := runes[i]
		current.WriteRune(r)

		// Check for sentence terminator (. ! ?) followed by whitespace or EOF
		if r == '.' || r == '!' || r == '?' {
			isNextWhitespace := (i+1 == length) || unicode.IsSpace(runes[i+1])
			if isNextWhitespace {
				s := strings.TrimSpace(current.String())
				if len(s) > 0 {
					sentences = append(sentences, s)
				}
				current.Reset()
			}
		}
	}

	remaining := strings.TrimSpace(current.String())
	if len(remaining) > 0 {
		sentences = append(sentences, remaining)
	}

	return sentences
}

func isBoilerplateSentence(s string) bool {
	lower := strings.ToLower(s)
	boilerplateWords := []string{
		"all rights reserved",
		"terms of service",
		"privacy policy",
		"cookie policy",
		"subscribe to our newsletter",
		"sign up for free",
		"click here to",
		"read full article",
		"leave a comment",
		"share on facebook",
		"share on twitter",
		"follow us on",
	}

	for _, bw := range boilerplateWords {
		if strings.Contains(lower, bw) {
			return true
		}
	}
	return false
}

func truncateCleanly(s string, maxChars int) string {
	if maxChars <= 0 {
		return ""
	}
	runes := []rune(s)
	if len(runes) <= maxChars {
		return s
	}

	target := string(runes[:maxChars])
	// Try to find the last sentence terminator before maxChars
	lastPeriod := strings.LastIndexAny(target, ".!?\n")
	if lastPeriod >= maxChars/2 {
		return strings.TrimSpace(target[:lastPeriod+1])
	}

	// Fallback to last whitespace
	lastSpace := strings.LastIndexFunc(target, unicode.IsSpace)
	if lastSpace > 0 {
		prefix := strings.TrimSpace(target[:lastSpace])
		if len([]rune(prefix))+2 <= maxChars {
			return prefix + " …"
		}
		return string([]rune(prefix)[:maxChars-1]) + "…"
	}

	if maxChars == 1 {
		return "…"
	}
	return strings.TrimSpace(string([]rune(target)[:maxChars-1])) + "…"
}
