package brave

import (
	"bytes"
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"

	"plewtus/internal/model"
	"plewtus/internal/provider"
)

func readFixture(t *testing.T, name string) []byte {
	t.Helper()
	data, err := os.ReadFile("../../../testdata/fixtures/" + name)
	if err != nil {
		t.Fatalf("failed to read fixture %s: %v", name, err)
	}
	return data
}

func TestParseHTMLResponseFixture(t *testing.T) {
	results, err := ParseHTMLResponse(bytes.NewReader(readFixture(t, "brave.html")))
	if err != nil {
		t.Fatalf("parse failed: %v", err)
	}

	wantURLs := []string{
		"https://go.dev/doc/devel/release",
		"https://go.dev/doc/go1.26",
		"https://go.dev/doc/go1.27",
		"https://go.dev/blog/go1.27",
	}
	if len(results) != len(wantURLs) {
		t.Fatalf("expected %d web results (video cluster and site links skipped), got %d: %+v", len(wantURLs), len(results), results)
	}
	for i, want := range wantURLs {
		if results[i].URL != want {
			t.Errorf("result %d: url %q, want %q", i, results[i].URL, want)
		}
		if results[i].Position != i+1 {
			t.Errorf("result %d: position %d, want %d", i, results[i].Position, i+1)
		}
	}

	if results[0].Title != "Release History - The Go Programming Language" {
		t.Errorf("unexpected title %q", results[0].Title)
	}
	if results[0].Snippet != "Go 1.26.0 is a major release of Go. Read the Go 1.26 Release Notes for more information." {
		t.Errorf("unexpected snippet %q", results[0].Snippet)
	}
	if results[0].PublishedDate != "" {
		t.Errorf("undated result got date %q", results[0].PublishedDate)
	}

	if results[2].PublishedDate != "3 weeks ago" || strings.HasPrefix(results[2].Snippet, "3 weeks") {
		t.Errorf("relative date not split from snippet: date=%q snippet=%q", results[2].PublishedDate, results[2].Snippet)
	}
	if results[3].PublishedDate != "19 August 2026" {
		t.Errorf("unexpected date %q", results[3].PublishedDate)
	}
	if !strings.HasPrefix(results[3].Snippet, "Go 1.27 adds generic methods") {
		t.Errorf("unexpected snippet %q", results[3].Snippet)
	}
}

func TestCaptchaPageDetected(t *testing.T) {
	body := readFixture(t, "brave_captcha.html")
	if !isCaptcha(http.StatusOK, body) {
		t.Error("real captcha page not recognised from its content")
	}
	if !isCaptcha(http.StatusTooManyRequests, nil) {
		t.Error("HTTP 429 must count as a captcha")
	}
	if isCaptcha(http.StatusOK, readFixture(t, "brave.html")) {
		t.Error("a results page must not count as a captcha")
	}
}

func newTestProvider(srv *httptest.Server, key string) *Provider {
	p := New(2*time.Second, "test-agent", key)
	p.htmlURL = srv.URL + "/search"
	p.apiURL = srv.URL + "/res/v1/web/search"
	return p
}

func TestHTMLSearchSendsQueryAndPreferences(t *testing.T) {
	fixture := readFixture(t, "brave.html")
	var got *http.Request
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		got = r
		_, _ = w.Write(fixture)
	}))
	defer srv.Close()

	resp, err := newTestProvider(srv, "").Search(context.Background(), model.ProviderRequest{
		Query: "golang release notes", Limit: 3, Freshness: "month", Country: "uk", Language: "en",
	})
	if err != nil {
		t.Fatalf("search failed: %v", err)
	}
	if resp.Engine != "brave" || len(resp.Results) != 3 {
		t.Fatalf("expected 3 brave results, got %s/%d", resp.Engine, len(resp.Results))
	}
	q := got.URL.Query()
	if q.Get("q") != "golang release notes" || q.Get("source") != "web" || q.Get("tf") != "pm" || q.Has("offset") {
		t.Errorf("unexpected query %v", q)
	}
	cookie := got.Header.Get("Cookie")
	for _, want := range []string{"country=gb", "ui_lang=en-gb", "safesearch=moderate"} {
		if !strings.Contains(cookie, want) {
			t.Errorf("cookie %q missing %q", cookie, want)
		}
	}
	if got.Header.Get("X-Subscription-Token") != "" {
		t.Error("HTML mode must not send an API key")
	}
}

func TestHTMLCaptchaIsReported(t *testing.T) {
	captcha := readFixture(t, "brave_captcha.html")
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusTooManyRequests)
		_, _ = w.Write(captcha)
	}))
	defer srv.Close()

	_, err := newTestProvider(srv, "").Search(context.Background(), model.ProviderRequest{Query: "x"})
	if !errors.Is(err, ErrChallengeEncountered) || !strings.Contains(err.Error(), "PLEWTUS_BRAVE_API_KEY") {
		t.Fatalf("expected a captcha error pointing at the API key, got %v", err)
	}
}

const apiFixture = `{"type":"search","query":{"original":"golang"},"web":{"type":"search","results":[
 {"title":"The <strong>Go</strong> Programming Language","url":"https://go.dev/","description":"<strong>Go</strong> is an open source programming language.","page_age":"2026-10-01T00:00:00"},
 {"title":"Go (programming language) - Wikipedia","url":"https://en.wikipedia.org/wiki/Go_(programming_language)","description":"Go is a statically typed language.","age":"2 days ago"},
 {"title":"not http","url":"ftp://example.org/","description":"skipped"}
]}}`

func TestAPISearchUsesKeyAndParses(t *testing.T) {
	var got *http.Request
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		got = r
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(apiFixture))
	}))
	defer srv.Close()

	resp, err := newTestProvider(srv, "secret").Search(context.Background(), model.ProviderRequest{
		Query: "golang", Limit: 10, Freshness: "week", IncludeDomains: []string{"go.dev"},
	})
	if err != nil {
		t.Fatalf("search failed: %v", err)
	}
	if got.URL.Path != "/res/v1/web/search" || got.Header.Get("X-Subscription-Token") != "secret" {
		t.Fatalf("API mode not used: path=%s token=%q", got.URL.Path, got.Header.Get("X-Subscription-Token"))
	}
	q := got.URL.Query()
	if q.Get("q") != "golang site:go.dev" || q.Get("count") != "20" || q.Get("freshness") != "pw" ||
		q.Get("country") != "US" || q.Get("search_lang") != "en" {
		t.Errorf("unexpected API query %v", q)
	}
	if len(resp.Results) != 2 {
		t.Fatalf("expected 2 http results, got %+v", resp.Results)
	}
	first := resp.Results[0]
	if first.Title != "The Go Programming Language" || first.Snippet != "Go is an open source programming language." {
		t.Errorf("highlight tags not stripped: %+v", first)
	}
	if first.PublishedDate != "2026-10-01T00:00:00" || resp.Results[1].PublishedDate != "2 days ago" {
		t.Errorf("unexpected dates: %q / %q", first.PublishedDate, resp.Results[1].PublishedDate)
	}
}

func TestAPIStatusErrors(t *testing.T) {
	cases := map[int]string{
		http.StatusTooManyRequests:     "rate limited",
		http.StatusUnauthorized:        "rejected the API key",
		http.StatusUnprocessableEntity: "request parameters",
		http.StatusInternalServerError: "unexpected status",
	}
	for status, want := range cases {
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(status)
		}))
		_, err := newTestProvider(srv, "k").Search(context.Background(), model.ProviderRequest{Query: "x"})
		srv.Close()
		if err == nil || !strings.Contains(err.Error(), want) {
			t.Errorf("status %d: expected %q, got %v", status, want, err)
		}
	}
}

func TestUnsupportedFreshness(t *testing.T) {
	_, err := New(time.Second, "", "").Search(context.Background(), model.ProviderRequest{Query: "x", Freshness: "decade"})
	if !errors.Is(err, provider.ErrUnsupportedFilter) {
		t.Fatalf("expected ErrUnsupportedFilter, got %v", err)
	}
}
