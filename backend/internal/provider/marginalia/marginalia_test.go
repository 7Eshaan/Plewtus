package marginalia

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

func readFixture(t *testing.T) []byte {
	t.Helper()
	data, err := os.ReadFile("../../../testdata/fixtures/marginalia.json")
	if err != nil {
		t.Fatalf("failed to read Marginalia fixture: %v", err)
	}
	return data
}

func TestParseResponseFixture(t *testing.T) {
	results, err := ParseResponse(readFixture(t))
	if err != nil {
		t.Fatalf("ParseResponse failed: %v", err)
	}
	if len(results) != 3 {
		t.Fatalf("expected 3 http(s) results (gopher skipped), got %d: %+v", len(results), results)
	}
	if results[0].URL != "https://go.dev/blog/pipelines" || results[0].Position != 1 {
		t.Errorf("unexpected first result: %+v", results[0])
	}
	if results[1].Snippet != "Go's built-in net/http package is convenient, solid and performant, making it easy to write production-grade web servers." {
		t.Errorf("snippet whitespace not normalized: %q", results[1].Snippet)
	}
	if results[2].Position != 3 {
		t.Errorf("positions must stay contiguous after skipping entries, got %d", results[2].Position)
	}
	if results[2].Title != results[2].URL {
		t.Errorf("empty title should fall back to the URL, got %q", results[2].Title)
	}
}

func TestParseResponseRejectsNonJSON(t *testing.T) {
	if _, err := ParseResponse([]byte("QPM Limit Exceeded")); err == nil {
		t.Fatal("expected an error for a non-JSON body")
	}
}

func newTestProvider(srv *httptest.Server, key string) *Provider {
	p := New(2*time.Second, "test-agent", key)
	p.baseURL = srv.URL + "/search"
	p.now = func() time.Time { return time.Date(2026, 11, 11, 0, 0, 0, 0, time.UTC) }
	return p
}

func TestSearchSendsKeyQueryAndCount(t *testing.T) {
	fixture := readFixture(t)
	var got *http.Request
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		got = r
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write(fixture)
	}))
	defer srv.Close()

	p := newTestProvider(srv, "")
	resp, err := p.Search(context.Background(), model.ProviderRequest{
		Query:          "golang concurrency",
		Limit:          2,
		Language:       "EN",
		Freshness:      "year",
		IncludeDomains: []string{"go.dev"},
	})
	if err != nil {
		t.Fatalf("Search failed: %v", err)
	}
	if resp.Engine != "marginalia" || len(resp.Results) != 2 {
		t.Fatalf("expected 2 marginalia results (trimmed to limit), got %s/%d", resp.Engine, len(resp.Results))
	}
	if key := got.Header.Get("API-Key"); key != PublicAPIKey {
		t.Errorf("expected the public key by default, got %q", key)
	}
	q := got.URL.Query()
	if q.Get("query") != "golang concurrency site:*.go.dev year>2025" {
		t.Errorf("unexpected query %q", q.Get("query"))
	}
	if q.Get("count") != "2" {
		t.Errorf("expected count to follow the provider limit, got %q", q.Get("count"))
	}
	if q.Get("lang") != "en" {
		t.Errorf("expected lang=en, got %q", q.Get("lang"))
	}
}

func TestSearchMultipleIncludeDomainsLeftToPipeline(t *testing.T) {
	p := New(time.Second, "", "k")
	q, err := p.buildQuery(model.ProviderRequest{Query: "raft", IncludeDomains: []string{"a.com", "b.com"}, ExcludeDomains: []string{"c.com"}})
	if err != nil {
		t.Fatal(err)
	}
	if q != "raft" {
		t.Errorf("Marginalia has no OR operator; expected bare query, got %q", q)
	}
}

func TestSearchUnsupportedFreshness(t *testing.T) {
	called := false
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { called = true }))
	defer srv.Close()

	_, err := newTestProvider(srv, "k").Search(context.Background(), model.ProviderRequest{Query: "news", Freshness: "week"})
	if !errors.Is(err, provider.ErrUnsupportedFilter) {
		t.Fatalf("expected ErrUnsupportedFilter, got %v", err)
	}
	if called {
		t.Error("an unsupported filter must not reach the API")
	}
}

func TestSearchStatusErrors(t *testing.T) {
	cases := []struct {
		status int
		key    string
		want   string
	}{
		{http.StatusTooManyRequests, "", "PLEWTUS_MARGINALIA_API_KEY"},
		{http.StatusTooManyRequests, "private", "status 429"},
		{http.StatusUnauthorized, "bad", "rejected the API key"},
		{http.StatusGatewayTimeout, "k", "timed out"},
		{http.StatusInternalServerError, "k", "unexpected status"},
	}
	for _, tc := range cases {
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(tc.status)
			_, _ = w.Write([]byte("QPM Limit Exceeded"))
		}))
		_, err := newTestProvider(srv, tc.key).Search(context.Background(), model.ProviderRequest{Query: "x"})
		srv.Close()
		if err == nil || !strings.Contains(err.Error(), tc.want) {
			t.Errorf("status %d key %q: expected error containing %q, got %v", tc.status, tc.key, tc.want, err)
		}
		if tc.key == "private" && strings.Contains(err.Error(), "PLEWTUS_MARGINALIA_API_KEY") {
			t.Errorf("the public-key hint must only appear for the public key")
		}
	}
}
