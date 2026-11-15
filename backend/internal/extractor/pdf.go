package extractor

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"html"
	"io"
	"os"
	"os/exec"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"time"
	"unicode"
	"unicode/utf8"
)

// PDF support: text is extracted by Poppler's pdftotext in a separate process
// (a malformed or hostile PDF cannot crash or hang the server: the process is
// killed when the extraction deadline passes), cleaned up (hyphenation, page
// headers/footers, page numbers, ligatures), and turned into a minimal HTML
// article so the normal content pipeline (passage selection, highlights,
// Markdown) applies to it unchanged.

var (
	// ErrPDFUnavailable reports that this server cannot read PDFs.
	ErrPDFUnavailable = errors.New("PDF extraction is not available on this server (pdftotext from poppler-utils is not installed)")
	// ErrPDFNoText reports a PDF without a text layer.
	ErrPDFNoText = errors.New("the PDF has no text layer (likely a scanned document; OCR is not supported)")
)

const (
	// pdfMaxPages bounds how much of a long document is converted
	// (pdftotext handles hundreds of pages in well under a second).
	pdfMaxPages = 150
	// pdfMaxTextBytes caps the text read back from pdftotext.
	pdfMaxTextBytes = 4 << 20
)

// pdftotextBinary locates pdftotext once; tests may override it.
var pdftotextBinary = sync.OnceValue(func() string {
	path, err := exec.LookPath("pdftotext")
	if err != nil {
		return ""
	}
	return path
})

// PDFSupported reports whether this server can extract PDFs.
func PDFSupported() bool { return pdftotextBinary() != "" }

// parsePDF extracts a PDF into a ParsedPage.
func parsePDF(ctx context.Context, data []byte) (*ParsedPage, error) {
	bin := pdftotextBinary()
	if bin == "" {
		return nil, ErrPDFUnavailable
	}

	// pdftotext needs a seekable file.
	tmp, err := os.CreateTemp("", "plewtus-*.pdf")
	if err != nil {
		return nil, fmt.Errorf("pdf extraction failed: %w", err)
	}
	defer os.Remove(tmp.Name())
	if _, err := tmp.Write(data); err != nil {
		tmp.Close()
		return nil, fmt.Errorf("pdf extraction failed: %w", err)
	}
	if err := tmp.Close(); err != nil {
		return nil, fmt.Errorf("pdf extraction failed: %w", err)
	}

	out := &cappedBuffer{limit: pdfMaxTextBytes}
	cmd := exec.CommandContext(ctx, bin, "-q", "-l", strconv.Itoa(pdfMaxPages), "-enc", "UTF-8", "-htmlmeta", tmp.Name(), "-")
	cmd.Stdout = out
	cmd.Stderr = io.Discard
	cmd.WaitDelay = time.Second
	if err := cmd.Run(); err != nil {
		if ctx.Err() != nil {
			return nil, ctx.Err()
		}
		return nil, fmt.Errorf("pdf extraction failed: the file could not be read (%v)", err)
	}

	meta, text := parsePDFToTextOutput(out.Bytes())
	paragraphs := pdfParagraphs(text)
	chars := 0
	for _, p := range paragraphs {
		chars += utf8.RuneCountInString(p.text)
	}
	if chars < minReadableChars {
		return nil, ErrPDFNoText
	}
	if meta.Title == "" || looksLikeFileName(meta.Title) {
		meta.Title = guessPDFTitle(paragraphs)
	}
	return ParseHTML(pdfHTML(meta, paragraphs))
}

// cappedBuffer keeps the first limit bytes and silently drops the rest, so
// the child process is never blocked or killed by a full pipe.
type cappedBuffer struct {
	bytes.Buffer
	limit int
}

func (b *cappedBuffer) Write(p []byte) (int, error) {
	if room := b.limit - b.Len(); room > 0 {
		if len(p) > room {
			b.Buffer.Write(p[:room])
		} else {
			b.Buffer.Write(p)
		}
	}
	return len(p), nil
}

var (
	pdfTitleTag = regexp.MustCompile(`(?is)<title>(.*?)</title>`)
	pdfMetaTag  = regexp.MustCompile(`(?is)<meta\s+name="([^"]+)"\s+content="([^"]*)"`)
	pdfPreBody  = regexp.MustCompile(`(?is)<pre>(.*)</pre>`)
	pdfDate     = regexp.MustCompile(`^(?:D:)?(\d{4})(\d{2})?(\d{2})?`)
)

// parsePDFToTextOutput reads `pdftotext -htmlmeta` output: document info in
// <head>, the text (page breaks as form feeds) HTML-escaped inside <pre>.
func parsePDFToTextOutput(out []byte) (PageMetadata, string) {
	s := string(out)
	var meta PageMetadata
	if m := pdfTitleTag.FindStringSubmatch(s); m != nil {
		meta.Title = strings.TrimSpace(html.UnescapeString(m[1]))
	}
	for _, m := range pdfMetaTag.FindAllStringSubmatch(s, -1) {
		value := strings.TrimSpace(html.UnescapeString(m[2]))
		switch strings.ToLower(m[1]) {
		case "author":
			meta.Author = value
		case "subject":
			meta.Description = value
		case "creationdate":
			meta.PublishedDate = formatPDFDate(value)
		}
	}
	text := s
	if m := pdfPreBody.FindStringSubmatch(s); m != nil {
		text = m[1]
	}
	return meta, html.UnescapeString(text)
}

// formatPDFDate turns "D:20240115093000Z" or "2024-01-15T..." into a date.
func formatPDFDate(value string) string {
	if strings.Contains(value, "-") {
		if len(value) >= 10 {
			return value[:10]
		}
		return value
	}
	m := pdfDate.FindStringSubmatch(value)
	if m == nil {
		return ""
	}
	date := m[1]
	if m[2] != "" {
		date += "-" + m[2]
		if m[3] != "" {
			date += "-" + m[3]
		}
	}
	return date
}

func looksLikeFileName(title string) bool {
	lower := strings.ToLower(title)
	for _, suffix := range []string{".pdf", ".doc", ".docx", ".tex", ".dvi", ".indd", ".ppt", ".pptx"} {
		if strings.HasSuffix(lower, suffix) {
			return true
		}
	}
	return strings.HasPrefix(lower, "microsoft word - ") || lower == "untitled"
}

// pdfParagraph is one paragraph of PDF text, or a section heading.
type pdfParagraph struct {
	text    string
	heading int // 2 or 3 for headings, else 0
}

var (
	pageNumberLine  = regexp.MustCompile(`(?i)^(page\s*)?\d{1,4}(\s*(of|/)\s*\d{1,4})?$`)
	numberedHeading = regexp.MustCompile(`^(\d{1,2}(\.\d{1,2})*\.?|[IVX]{1,5}\.|[A-Z]\.)\s+\p{Lu}`)
	digitRun        = regexp.MustCompile(`\d+`)
	ligatures       = strings.NewReplacer("ﬁ", "fi", "ﬂ", "fl", "ﬀ", "ff", "ﬃ", "ffi", "ﬄ", "ffl", "­", "")
)

var sectionNames = map[string]bool{
	"abstract": true, "introduction": true, "background": true, "related work": true, "method": true,
	"methods": true, "methodology": true, "materials and methods": true, "results": true, "discussion": true,
	"conclusion": true, "conclusions": true, "summary": true, "executive summary": true, "evaluation": true,
	"experiments": true, "limitations": true, "future work": true, "appendix": true,
	"acknowledgements": true, "acknowledgments": true, "foreword": true, "preface": true,
}

// endSections start the reference list, which is cut when it comes in the
// second half of the document (front matter can also be titled this way).
var endSections = map[string]bool{
	"references": true, "bibliography": true, "works cited": true,
}

var (
	// bareSectionNumber is a section number alone on its line ("3.2"); LaTeX
	// papers often put the title on the next line.
	bareSectionNumber = regexp.MustCompile(`^\d{1,2}(\.\d{1,2})*\.?$`)
	// arxivStamp is the identifier arXiv prints down the first page's margin.
	arxivStamp = regexp.MustCompile(`(?i)^arxiv:\d{4}\.\d{4,5}(v\d+)?\b`)
	bulletLine = regexp.MustCompile(`^[•▪◦‣∙·\-–] `)
)

// pdfParagraphs cleans pdftotext output into paragraphs: running headers,
// footers, and page numbers are dropped, lines are joined (re-joining words
// hyphenated across line breaks), likely section headings are marked, and
// everything from the reference list on is left out.
func pdfParagraphs(text string) []pdfParagraph {
	text = ligatures.Replace(strings.ReplaceAll(text, "\r\n", "\n"))
	pages := strings.Split(text, "\f")
	repeated := repeatedEdgeLines(pages)

	var paragraphs []pdfParagraph
	var cur []string
	flush := func() {
		if len(cur) == 0 {
			return
		}
		joined := joinPDFLines(cur)
		cur = nil
		if joined == "" {
			return
		}
		paragraphs = append(paragraphs, pdfParagraph{text: joined, heading: headingLevel(joined)})
	}

	pendingNumber := ""
	for _, page := range pages {
		lines := strings.Split(page, "\n")
		lastLine := -1
		for i, line := range lines {
			if strings.TrimSpace(line) != "" {
				lastLine = i
			}
		}
		for i, line := range lines {
			trimmed := strings.TrimSpace(line)
			if trimmed == "" {
				flush()
				continue
			}
			if pendingNumber != "" {
				number := pendingNumber
				pendingNumber = ""
				if candidate := number + " " + trimmed; headingLevel(candidate) > 0 {
					flush()
					cur = append(cur, candidate)
					flush()
					continue
				}
				cur = append(cur, number) // not a heading after all
			}
			switch {
			case repeated[edgeKey(trimmed)] || arxivStamp.MatchString(trimmed):
				// running header/footer or margin stamp
			case bareSectionNumber.MatchString(trimmed) && i != lastLine &&
				(strings.Contains(trimmed, ".") || !pageNumberLine.MatchString(trimmed) || i > 1):
				// "3.2" alone (or "3" away from the page edges): wait for the title.
				flush()
				pendingNumber = trimmed
			case pageNumberLine.MatchString(trimmed):
				// page number
			default:
				// A heading or a bullet starts a new paragraph.
				if len(cur) > 0 && (headingLevel(trimmed) > 0 || bulletLine.MatchString(trimmed)) {
					flush()
				}
				cur = append(cur, trimmed)
				if headingLevel(trimmed) > 0 {
					flush()
				}
			}
		}
		// A page break does not end a sentence that continues on the next page.
		if n := len(cur); n > 0 {
			if last, _ := utf8.DecodeLastRuneInString(cur[n-1]); strings.ContainsRune(".!?:", last) {
				flush()
			}
		}
	}
	flush()

	total := 0
	for _, p := range paragraphs {
		total += len(p.text)
	}
	seen := 0
	for i, p := range paragraphs {
		if p.heading > 0 && endSections[strings.ToLower(strings.TrimRight(stripSectionNumber(p.text), ":"))] && seen > total/2 {
			return paragraphs[:i]
		}
		seen += len(p.text)
	}
	return paragraphs
}

func joinPDFLines(lines []string) string {
	var b strings.Builder
	for i, line := range lines {
		if i == 0 {
			b.WriteString(line)
			continue
		}
		prev := b.String()
		first, _ := utf8.DecodeRuneInString(line)
		if strings.HasSuffix(prev, "-") && len(prev) > 1 && unicode.IsLower(first) {
			before, _ := utf8.DecodeLastRuneInString(prev[:len(prev)-1])
			if unicode.IsLetter(before) {
				// "informa-" + "tion" -> "information"
				b.Reset()
				b.WriteString(prev[:len(prev)-1])
				b.WriteString(line)
				continue
			}
		}
		b.WriteByte(' ')
		b.WriteString(line)
	}
	return strings.Join(strings.Fields(b.String()), " ")
}

func stripSectionNumber(s string) string {
	if loc := numberedHeading.FindStringIndex(s); loc != nil {
		// keep the first letter of the title, which the pattern consumed
		return strings.TrimSpace(s[loc[1]-1:])
	}
	return s
}

// headingLevel recognises section headings: numbered ("2.1 Results") or
// well-known section names, on a short line without closing punctuation.
func headingLevel(line string) int {
	if utf8.RuneCountInString(line) > 90 {
		return 0
	}
	if last, _ := utf8.DecodeLastRuneInString(line); strings.ContainsRune(".,;", last) {
		return 0
	}
	// Numbered rules and formulas ("5. If x > y, set z =") are not headings.
	if strings.ContainsAny(line, "=<>{}[]|;") || len(strings.Fields(line)) > 12 {
		return 0
	}
	if m := numberedHeading.FindString(line); m != "" {
		// "4. Append any new entries not already in the log" is a numbered
		// list item: headings are short or written in title case.
		if title := strings.Fields(line)[1:]; len(title) > 7 && capitalizedShare(title) < 0.6 {
			return 0
		}
		if strings.Count(strings.TrimRight(strings.Fields(m)[0], "."), ".") > 0 {
			return 3
		}
		return 2
	}
	name := strings.ToLower(strings.TrimRight(line, ":"))
	if sectionNames[name] || endSections[name] {
		return 2
	}
	return 0
}

func edgeKey(line string) string {
	return digitRun.ReplaceAllString(strings.ToLower(strings.Join(strings.Fields(line), " ")), "#")
}

// edgeLines is how many lines at the top and bottom of a page may hold a
// running header or footer (RFCs use three or four).
const edgeLines = 4

// repeatedEdgeLines finds lines that open or close most pages: running
// headers and footers ("Journal of X, Vol. 3", "Author et al.").
func repeatedEdgeLines(pages []string) map[string]bool {
	repeated := make(map[string]bool)
	if len(pages) < 3 {
		return repeated
	}
	counts := make(map[string]int)
	for _, page := range pages {
		var lines []string
		for _, l := range strings.Split(page, "\n") {
			if t := strings.TrimSpace(l); t != "" {
				lines = append(lines, t)
			}
		}
		seen := make(map[string]bool)
		for i, l := range lines {
			if i < edgeLines || i >= len(lines)-edgeLines {
				if k := edgeKey(l); !seen[k] {
					seen[k] = true
					counts[k]++
				}
			}
		}
	}
	threshold := len(pages) / 2
	if threshold < 2 {
		threshold = 2
	}
	for k, n := range counts {
		if n >= threshold && utf8.RuneCountInString(k) < 120 {
			repeated[k] = true
		}
	}
	return repeated
}

// guessPDFTitle picks the first title-like line among the opening
// paragraphs: a few words, mostly capitalized, no e-mail or final period
// (skipping licence notices and margin stamps that often come first).
func guessPDFTitle(paragraphs []pdfParagraph) string {
	for i, p := range paragraphs {
		if i >= 10 {
			break
		}
		if p.heading > 0 {
			continue // "1 Introduction" is a section, not the title
		}
		words := strings.Fields(p.text)
		if len(words) < 2 || len(words) > 20 || len(p.text) > 150 || strings.ContainsAny(p.text, "@") {
			continue
		}
		if last, _ := utf8.DecodeLastRuneInString(p.text); last == '.' || last == ',' {
			continue
		}
		if capitalizedShare(words) >= 0.5 {
			return p.text
		}
	}
	return ""
}

// pdfHTML renders the cleaned PDF as a minimal HTML article.
func pdfHTML(meta PageMetadata, paragraphs []pdfParagraph) []byte {
	var b strings.Builder
	b.WriteString("<!DOCTYPE html><html><head><meta charset=\"utf-8\">")
	if meta.Title != "" {
		b.WriteString("<title>" + html.EscapeString(meta.Title) + "</title>")
	}
	if meta.Author != "" {
		b.WriteString(`<meta name="author" content="` + html.EscapeString(meta.Author) + `">`)
	}
	if meta.Description != "" {
		b.WriteString(`<meta name="description" content="` + html.EscapeString(meta.Description) + `">`)
	}
	if meta.PublishedDate != "" {
		b.WriteString(`<meta name="date" content="` + html.EscapeString(meta.PublishedDate) + `">`)
	}
	b.WriteString("</head><body><article>")
	if meta.Title != "" {
		b.WriteString("<h1>" + html.EscapeString(meta.Title) + "</h1>")
	}
	for i, p := range paragraphs {
		if i == 0 && strings.EqualFold(p.text, meta.Title) {
			continue // the title line repeated as the first paragraph
		}
		tag := "p"
		if p.heading > 0 {
			tag = "h" + strconv.Itoa(p.heading)
		}
		b.WriteString("<" + tag + ">" + html.EscapeString(p.text) + "</" + tag + ">")
	}
	b.WriteString("</article></body></html>")
	return []byte(b.String())
}

// capitalizedShare is the fraction of words starting with a capital letter or
// a digit.
func capitalizedShare(words []string) float64 {
	if len(words) == 0 {
		return 0
	}
	capitalized := 0
	for _, w := range words {
		if first, _ := utf8.DecodeRuneInString(w); unicode.IsUpper(first) || unicode.IsDigit(first) {
			capitalized++
		}
	}
	return float64(capitalized) / float64(len(words))
}
