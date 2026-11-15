package extractor

import (
	"context"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"plewtus/internal/model"
)

func pdfResponse(contentType, body string) *mockRoundTripper {
	return &mockRoundTripper{roundTripFunc: func(req *http.Request) (*http.Response, error) {
		return &http.Response{
			StatusCode: http.StatusOK,
			Status:     "200 OK",
			Header:     http.Header{"Content-Type": []string{contentType}},
			Body:       io.NopCloser(strings.NewReader(body)),
		}, nil
	}}
}

func TestPDF_FetcherAcceptsRealPDFs(t *testing.T) {
	for _, contentType := range []string{"application/pdf", "application/octet-stream", ""} {
		fetcher := NewFetcher(FetcherOptions{Transport: pdfResponse(contentType, "%PDF-1.7\n binary")})
		body, got, err := fetcher.Fetch(context.Background(), "https://example.com/paper.pdf")
		if err != nil || got != PDFContentType || !strings.HasPrefix(string(body), "%PDF-") {
			t.Errorf("content type %q: expected a PDF, got %q, %v", contentType, got, err)
		}
	}
}

func TestPDF_FetcherRejectsFakePDFs(t *testing.T) {
	fetcher := NewFetcher(FetcherOptions{Transport: pdfResponse("application/pdf", "<html>Access denied</html>")})
	if _, _, err := fetcher.Fetch(context.Background(), "https://example.com/paper.pdf"); err == nil || !strings.Contains(err.Error(), "not a PDF") {
		t.Fatalf("expected a non-PDF body to be rejected, got %v", err)
	}
}

func TestPDF_FetcherCapsPDFSize(t *testing.T) {
	fetcher := NewFetcher(FetcherOptions{Transport: pdfResponse("application/pdf", "%PDF-1.7"+strings.Repeat("x", 4096)), MaxPDFBytes: 1024})
	if _, _, err := fetcher.Fetch(context.Background(), "https://example.com/big.pdf"); err == nil || !strings.Contains(err.Error(), "exceeds 1024 bytes") {
		t.Fatalf("expected the PDF size cap, got %v", err)
	}
}

// samplePDFText is pdftotext-style output: three pages separated by form
// feeds, with a running header, page numbers, a word hyphenated across a line
// break, numbered sections, and a reference list.
const samplePDFText = "Climate and Crops\n\nJournal of Agronomy, Vol. 12\n\n" +
	"Abstract\n\nWarming reduces wheat yields by about six percent per degree of global warming, accord-\ning to field trials across many regions and seasons.\n\n1\n\f" +
	"Journal of Agronomy, Vol. 12\n\n1 Introduction\n\nRising temperatures shorten the growing season and raise heat stress at flowering, which lowers grain number.\n\n" +
	"2.1 Field trials\n\nTrials in 40 sites compared yields between warm and cool years over two decades of records.\n\n2\n\f" +
	"Journal of Agronomy, Vol. 12\n\nConclusion\n\nAdaptation through earlier sowing and heat-tolerant varieties recovers part of the loss.\n\n" +
	"References\n\n[1] Smith J. Wheat and heat. Nature 2020.\n\n3\n"

func TestPDF_ParagraphCleanup(t *testing.T) {
	paragraphs := pdfParagraphs(samplePDFText)
	var all []string
	headings := map[string]int{}
	for _, p := range paragraphs {
		all = append(all, p.text)
		if p.heading > 0 {
			headings[p.text] = p.heading
		}
	}
	joined := strings.Join(all, "\n")
	if strings.Contains(joined, "Journal of Agronomy") {
		t.Errorf("running header should be dropped:\n%s", joined)
	}
	if strings.Contains(joined, "\n1\n") || strings.HasSuffix(joined, "\n3") {
		t.Errorf("page numbers should be dropped:\n%s", joined)
	}
	if !strings.Contains(joined, "according to field trials") {
		t.Errorf("hyphenated word should be re-joined:\n%s", joined)
	}
	if strings.Contains(joined, "Smith J.") {
		t.Errorf("reference list should be cut:\n%s", joined)
	}
	if headings["Abstract"] != 2 || headings["1 Introduction"] != 2 || headings["2.1 Field trials"] != 3 || headings["Conclusion"] != 2 {
		t.Errorf("headings not recognised: %v", headings)
	}
}

func TestPDF_ParsesPdftotextHTMLMeta(t *testing.T) {
	out := `<!DOCTYPE html><html><head><title>Climate &amp; Crops</title>
<meta name="Author" content="A. Researcher"/>
<meta name="CreationDate" content="D:20240115093000Z"/>
</head><body><pre>Body &lt;text&gt;
</pre></body></html>`
	meta, text := parsePDFToTextOutput([]byte(out))
	if meta.Title != "Climate & Crops" || meta.Author != "A. Researcher" || meta.PublishedDate != "2024-01-15" {
		t.Fatalf("unexpected metadata %+v", meta)
	}
	if !strings.Contains(text, "Body <text>") {
		t.Fatalf("unexpected text %q", text)
	}
}

// fakePdftotext installs a script that prints pdftotext -htmlmeta output.
func fakePdftotext(t *testing.T, body string) {
	t.Helper()
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "out.txt"), []byte(body), 0o600); err != nil {
		t.Fatal(err)
	}
	script := filepath.Join(dir, "pdftotext")
	if err := os.WriteFile(script, []byte("#!/bin/sh\ncat '"+filepath.Join(dir, "out.txt")+"'\n"), 0o700); err != nil {
		t.Fatal(err)
	}
	previous := pdftotextBinary
	pdftotextBinary = func() string { return script }
	t.Cleanup(func() { pdftotextBinary = previous })
}

func TestPDF_ExtractEndToEnd(t *testing.T) {
	fakePdftotext(t, `<html><head><title>Climate and Crops</title><meta name="Author" content="A. Researcher"/></head><body><pre>`+
		samplePDFText+`</pre></body></html>`)
	ext := New(ExtractorOptions{Fetcher: NewFetcher(FetcherOptions{Transport: pdfResponse("application/pdf", "%PDF-1.7 ...")})})
	content, err := ext.Extract(context.Background(), "https://example.com/paper.pdf",
		&model.ExtractOptions{Mode: "markdown", MaxCharsPerPage: 4000, Query: "wheat yield per degree"})
	if err != nil {
		t.Fatal(err)
	}
	if content.Title != "Climate and Crops" || content.Author != "A. Researcher" {
		t.Errorf("metadata not carried over: %+v", content)
	}
	for _, want := range []string{"# Climate and Crops", "## Abstract", "six percent per degree", "### 2.1 Field trials"} {
		if !strings.Contains(content.Markdown, want) {
			t.Errorf("missing %q in:\n%s", want, content.Markdown)
		}
	}
}

func TestPDF_ScannedPDFHasNoTextLayer(t *testing.T) {
	fakePdftotext(t, "<html><head></head><body><pre>\f\f\f</pre></body></html>")
	ext := New(ExtractorOptions{Fetcher: NewFetcher(FetcherOptions{Transport: pdfResponse("application/pdf", "%PDF-1.7 ...")})})
	_, err := ext.Extract(context.Background(), "https://example.com/scan.pdf", &model.ExtractOptions{Mode: "text", MaxCharsPerPage: 4000})
	if !errors.Is(err, ErrPDFNoText) {
		t.Fatalf("expected ErrPDFNoText, got %v", err)
	}
}

func TestPDF_UnavailableWithoutPdftotext(t *testing.T) {
	previous := pdftotextBinary
	pdftotextBinary = func() string { return "" }
	t.Cleanup(func() { pdftotextBinary = previous })
	ext := New(ExtractorOptions{Fetcher: NewFetcher(FetcherOptions{Transport: pdfResponse("application/pdf", "%PDF-1.7 ...")})})
	_, err := ext.Extract(context.Background(), "https://example.com/paper.pdf", &model.ExtractOptions{Mode: "text", MaxCharsPerPage: 4000})
	if !errors.Is(err, ErrPDFUnavailable) {
		t.Fatalf("expected ErrPDFUnavailable, got %v", err)
	}
}

func TestPDF_RealWorldLayouts(t *testing.T) {
	body := strings.Repeat("Body text about consensus that continues for a while across the page. ", 3)
	text := "Provided proper attribution is provided, Google hereby grants permission to reproduce the tables.\n\n" +
		"arXiv:1706.03762v7 [cs.CL] 2 Aug 2023\n\n" +
		"Attention Is All You Need\n\n" +
		"Acknowledgments\n\nWe thank the reviewers.\n\n" + // front matter: must not cut the paper
		"Contents\n\n" + body + "\n\f" +
		"Intro text " + body + "\nthe end of a sentence.\n3.2\n\nScaled Dot-Product Attention\n\n" + body + "\n" +
		"5. If leaderCommit > commitIndex, set commitIndex =\n" + body + "\n" +
		"• Leader election: Raft uses randomized timers.\n• Membership changes: joint consensus.\n" +
		"4. Append any new entries not already in the log\n\n3.6 Backup Tasks\n\n" + body + "\n\f" +
		strings.Repeat(body+"\n\n", 6) + "References\n\n[1] Some citation.\n"
	paragraphs := pdfParagraphs(text)
	headings := map[string]int{}
	var all []string
	for _, p := range paragraphs {
		all = append(all, p.text)
		if p.heading > 0 {
			headings[p.text] = p.heading
		}
	}
	joined := strings.Join(all, "\n")
	if headings["3.2 Scaled Dot-Product Attention"] != 3 {
		t.Errorf("section number on its own line not joined to its title: %v", headings)
	}
	if _, ok := headings["5. If leaderCommit > commitIndex, set commitIndex ="]; ok {
		t.Errorf("a numbered rule was taken for a heading")
	}
	if _, ok := headings["4. Append any new entries not already in the log"]; ok {
		t.Errorf("a numbered list item was taken for a heading")
	}
	if headings["3.6 Backup Tasks"] != 3 {
		t.Errorf("short numbered heading lost: %v", headings)
	}
	if strings.Contains(joined, "arXiv:1706") {
		t.Errorf("arXiv margin stamp should be dropped")
	}
	if !strings.Contains(joined, "Scaled Dot-Product Attention") {
		t.Errorf("front-matter Acknowledgments cut the document:\n%s", joined)
	}
	if strings.Contains(joined, "Some citation") {
		t.Errorf("reference list at the end should be cut")
	}
	if !strings.Contains(joined, "\n• Membership changes") {
		t.Errorf("bullets should start their own paragraphs:\n%s", joined)
	}
	if got := guessPDFTitle(paragraphs); got != "Attention Is All You Need" {
		t.Errorf("title guess = %q", got)
	}
}

func TestPDF_SlowPDFsGetMoreTimeThanPages(t *testing.T) {
	slow := func(contentType, start string) http.HandlerFunc {
		return func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", contentType)
			_, _ = w.Write([]byte(start))
			w.(http.Flusher).Flush()
			time.Sleep(300 * time.Millisecond) // longer than the page timeout
			_, _ = w.Write([]byte(strings.Repeat(" more", 100)))
		}
	}
	fetch := func(handler http.HandlerFunc) error {
		ts := httptest.NewServer(handler)
		defer ts.Close()
		f := NewFetcher(FetcherOptions{Timeout: 100 * time.Millisecond, Transport: rewriteTransport{target: ts.URL, base: ts.Client().Transport}})
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		_, _, err := f.Fetch(ctx, "http://example.com/doc")
		return err
	}
	if err := fetch(slow("text/html", "<html><body><p>")); !errors.Is(err, context.DeadlineExceeded) {
		t.Errorf("a slow web page should time out, got %v", err)
	}
	if err := fetch(slow("application/pdf", "%PDF-1.7\n")); err != nil {
		t.Errorf("a slow PDF within the caller's deadline should load, got %v", err)
	}
}
