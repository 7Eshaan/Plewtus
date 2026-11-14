package qwant

import (
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

// The success fixture follows the documented v3 response shape (a real one
// could not be captured: Qwant is geo-blocked from where this was written).
// The error bodies below are real replies.
const (
	realNotInCountry = `{"status":"error","data":{"error_code":50}}`
	captchaBody      = `{"url":"https://geo.captcha-delivery.com/captcha/?initialCid=abc"}`
)

func readFixture(t *testing.T) []byte {
	t.Helper()
	data, err := os.ReadFile("../../../testdata/fixtures/qwant.json")
	if err != nil {
		t.Fatalf("failed to read fixture: %v", err)
	}
	return data
}

func TestParseResponseFixture(t *testing.T) {
	parsed, err := ParseResponse(readFixture(t))
	if err != nil {
		t.Fatalf("parse failed: %v", err)
	}
	r := parsed.Results
	if len(r) != 3 {
		t.Fatalf("expected 3 web results (ads, videos and non-http skipped), got %d: %+v", len(r), r)
	}
	if r[0].Title != "Concurrency is not parallelism" || r[0].Snippet != "Rob Pike on Go concurrency: concurrency is not parallelism." || r[0].PublishedDate != "2013-01-02" {
		t.Errorf("unexpected first result: %+v", r[0])
	}
	if r[1].Title != r[1].URL || r[1].PublishedDate != "" {
		t.Errorf("empty title should fall back to the URL and zero date be empty: %+v", r[1])
	}
	if r[2].URL != "https://go.dev/doc/effective_go#concurrency" || r[2].Position != 3 || r[2].PublishedDate != "2023-11-14" {
		t.Errorf("second web row or millisecond date not handled: %+v", r[2])
	}
}

func TestParseResponseErrors(t *testing.T) {
	parsed, err := ParseResponse([]byte(realNotInCountry))
	if err != nil || parsed.ErrorCode != errCodeNotInCountry {
		t.Errorf("geo-block not decoded: %+v, %v", parsed, err)
	}
	parsed, err = ParseResponse([]byte(captchaBody))
	if err != nil || !parsed.Captcha {
		t.Errorf("captcha not decoded: %+v, %v", parsed, err)
	}
}

func newTestProvider(handler http.HandlerFunc) (*Provider, func()) {
	srv := httptest.NewServer(handler)
	p := New(2*time.Second, "test-agent")
	p.baseURL = srv.URL + "/v3/search/web"
	return p, srv.Close
}

func TestSearchSendsQueryAndPages(t *testing.T) {
	fixture := readFixture(t)
	var offsets []string
	var first *http.Request
	p, stop := newTestProvider(func(w http.ResponseWriter, r *http.Request) {
		if first == nil {
			first = r
		}
		offsets = append(offsets, r.URL.Query().Get("offset"))
		_, _ = w.Write(fixture)
	})
	defer stop()

	resp, err := p.Search(context.Background(), model.ProviderRequest{
		Query: "golang concurrency", Limit: 2, Language: "fr", IncludeDomains: []string{"go.dev"},
	})
	if err != nil {
		t.Fatalf("search failed: %v", err)
	}
	if resp.Engine != "qwant" || len(resp.Results) != 2 {
		t.Fatalf("expected 2 qwant results, got %s/%d", resp.Engine, len(resp.Results))
	}
	q := first.URL.Query()
	if q.Get("q") != "golang concurrency site:go.dev" || q.Get("locale") != "fr_FR" || q.Get("count") != "10" || q.Get("device") != "desktop" {
		t.Errorf("unexpected query %v", q)
	}
	if strings.Join(offsets, ",") != "0" {
		t.Errorf("expected a single page at offset 0, got %v", offsets)
	}
}

func TestSearchReportsQwantFailures(t *testing.T) {
	cases := []struct {
		name   string
		status int
		body   string
		want   error
		text   string
	}{
		{"geo-blocked (real reply)", http.StatusForbidden, realNotInCountry, ErrNotAvailable, "country"},
		{"captcha", http.StatusForbidden, captchaBody, ErrChallengeEncountered, "DataDome"},
		{"blocked without json", http.StatusForbidden, "<html>blocked</html>", ErrChallengeEncountered, "403"},
		{"rate limited", http.StatusOK, `{"status":"error","data":{"error_code":24}}`, nil, "rate limited"},
		{"other api error", http.StatusOK, `{"status":"error","data":{"error_code":7}}`, nil, "error_code 7"},
		{"server error", http.StatusBadGateway, "bad gateway", nil, "unexpected status"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			p, stop := newTestProvider(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(tc.status)
				_, _ = w.Write([]byte(tc.body))
			})
			defer stop()
			_, err := p.Search(context.Background(), model.ProviderRequest{Query: "x"})
			if err == nil || !strings.Contains(err.Error(), tc.text) {
				t.Fatalf("expected error containing %q, got %v", tc.text, err)
			}
			if tc.want != nil && !errors.Is(err, tc.want) {
				t.Errorf("expected %v, got %v", tc.want, err)
			}
		})
	}
}

func TestFreshnessIsUnsupported(t *testing.T) {
	called := false
	p, stop := newTestProvider(func(w http.ResponseWriter, r *http.Request) { called = true })
	defer stop()
	_, err := p.Search(context.Background(), model.ProviderRequest{Query: "x", Freshness: "week"})
	if !errors.Is(err, provider.ErrUnsupportedFilter) || called {
		t.Fatalf("expected ErrUnsupportedFilter without a request, got %v (called=%v)", err, called)
	}
}

func TestLocale(t *testing.T) {
	cases := map[[2]string]string{
		{"", ""}:     "en_US",
		{"en", "uk"}: "en_GB",
		{"de", ""}:   "de_DE",
		{"ja", ""}:   "ja_US",
		{"fr", "ca"}: "fr_CA",
	}
	for in, want := range cases {
		if got := Locale(in[0], in[1]); got != want {
			t.Errorf("Locale(%q, %q) = %q, want %q", in[0], in[1], got, want)
		}
	}
}
