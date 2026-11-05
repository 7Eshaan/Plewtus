package extractor

import (
	"fmt"
	"sort"
	"strings"
	"unicode"

	"golang.org/x/net/html"
	"golang.org/x/net/html/atom"
)

// ToMarkdown converts an HTML node and its subtree into clean, formatted Markdown.
func ToMarkdown(n *html.Node, maxChars int) string {
	var buf strings.Builder
	renderMarkdownNode(n, &buf, &markdownState{})

	raw := buf.String()
	cleaned := cleanMarkdownOutput(raw)

	if maxChars > 0 && len(cleaned) > maxChars {
		return truncateCleanly(cleaned, maxChars)
	}

	return cleaned
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
		case atom.H1:
			buf.WriteString("\n\n# ")
			renderChildren(n, buf, state)
			buf.WriteString("\n\n")
			return
		case atom.H2:
			buf.WriteString("\n\n## ")
			renderChildren(n, buf, state)
			buf.WriteString("\n\n")
			return
		case atom.H3:
			buf.WriteString("\n\n### ")
			renderChildren(n, buf, state)
			buf.WriteString("\n\n")
			return
		case atom.H4, atom.H5, atom.H6:
			buf.WriteString("\n\n#### ")
			renderChildren(n, buf, state)
			buf.WriteString("\n\n")
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
			src := strings.TrimSpace(getAttr(n, "src"))
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
		case atom.Strong, atom.B:
			buf.WriteString("**")
			renderChildren(n, buf, state)
			buf.WriteString("**")
			return
		case atom.Em, atom.I:
			buf.WriteString("*")
			renderChildren(n, buf, state)
			buf.WriteString("*")
			return
		case atom.A:
			href := strings.TrimSpace(getAttr(n, "href"))
			var linkTextBuf strings.Builder
			renderChildren(n, &linkTextBuf, state)
			text := strings.TrimSpace(linkTextBuf.String())

			if text == "" {
				return
			}
			if href != "" && !strings.HasPrefix(strings.ToLower(href), "javascript:") {
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
		text := normalizeInlineWhitespace(n.Data)
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

func normalizeInlineWhitespace(s string) string {
	fields := strings.Fields(s)
	return strings.Join(fields, " ")
}

func cleanMarkdownOutput(s string) string {
	lines := strings.Split(s, "\n")
	var cleaned []string
	consecutiveBlank := 0

	for _, line := range lines {
		trimmed := strings.TrimRightFunc(line, unicode.IsSpace)
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
