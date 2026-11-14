package test

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	"plewtus/internal/config"
	"plewtus/internal/model"
	"plewtus/internal/orchestrator"
	"plewtus/internal/provider"
	"plewtus/internal/provider/brave"
	"plewtus/internal/provider/qwant"
	"plewtus/internal/server"
)

// fixedProvider replays results parsed from a captured engine response.
type fixedProvider struct {
	name    string
	results []model.RawSearchResult
}

func (p *fixedProvider) Name() string { return p.name }
func (p *fixedProvider) Search(context.Context, model.ProviderRequest) (*model.ProviderResponse, error) {
	return &model.ProviderResponse{Engine: p.name, Results: p.results, Took: time.Millisecond}, nil
}

func searchVia(t *testing.T, reg *provider.Registry, body string) model.SearchResponse {
	t.Helper()
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	cfg := config.Load()
	cfg.AuthRequired = false
	srv := server.New(cfg, logger, orchestrator.New(reg, time.Second, "plewtus-test", logger))

	req := httptest.NewRequest(http.MethodPost, "/search", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	srv.Handler().ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rec.Code, rec.Body.String())
	}
	var resp model.SearchResponse
	if err := json.NewDecoder(rec.Body).Decode(&resp); err != nil {
		t.Fatal(err)
	}
	return resp
}

// Real Brave results (the captured page) survive the pipeline and merge with
// another engine's copy of the same page.
func TestE2E_BraveResultsFlowThroughPipeline(t *testing.T) {
	page, err := os.ReadFile("../testdata/fixtures/brave.html")
	if err != nil {
		t.Fatal(err)
	}
	parsed, err := brave.ParseHTMLResponse(bytes.NewReader(page))
	if err != nil || len(parsed) == 0 {
		t.Fatalf("fixture did not parse: %v", err)
	}

	reg := provider.NewRegistry()
	reg.Register(&fixedProvider{name: "brave", results: parsed})
	reg.Register(&fixedProvider{name: "duckduckgo", results: []model.RawSearchResult{
		{Title: "Go 1.26 Release Notes", URL: "https://go.dev/doc/go1.26?utm_source=ddg", Snippet: "Go 1.26 release notes.", Position: 1},
	}})

	resp := searchVia(t, reg, `{"query":"go release notes","limit":10,"providers":["brave","duckduckgo"]}`)
	if len(resp.Metadata.Providers.Failed) != 0 {
		t.Fatalf("unexpected failures: %+v", resp.Metadata.Providers.Failed)
	}
	if len(resp.Results) != len(parsed) {
		t.Fatalf("expected %d merged results, got %d: %+v", len(parsed), len(resp.Results), resp.Results)
	}

	top := resp.Results[0]
	if top.URL != "https://go.dev/doc/go1.26" || len(top.Engines) != 2 {
		t.Errorf("the page both engines found should rank first with both engines credited, got %+v", top)
	}
	for _, r := range resp.Results {
		if r.Title == "" || r.URL == "" {
			t.Errorf("incomplete result: %+v", r)
		}
		if r.URL == "https://go.dev/blog/go1.27" && r.PublishedDate != "19 August 2026" {
			t.Errorf("published date lost: %+v", r)
		}
	}
}

// Qwant results (documented response shape) survive the pipeline.
func TestE2E_QwantResultsFlowThroughPipeline(t *testing.T) {
	body, err := os.ReadFile("../testdata/fixtures/qwant.json")
	if err != nil {
		t.Fatal(err)
	}
	parsed, err := qwant.ParseResponse(body)
	if err != nil || len(parsed.Results) == 0 {
		t.Fatalf("fixture did not parse: %v", err)
	}

	reg := provider.NewRegistry()
	reg.Register(&fixedProvider{name: "qwant", results: parsed.Results})

	resp := searchVia(t, reg, `{"query":"golang concurrency","limit":10,"providers":["qwant"]}`)
	if len(resp.Results) != len(parsed.Results) {
		t.Fatalf("expected %d results, got %d: %+v", len(parsed.Results), len(resp.Results), resp.Results)
	}
	for _, r := range resp.Results {
		if len(r.Engines) != 1 || r.Engines[0].Name != "qwant" {
			t.Errorf("result not attributed to qwant: %+v", r)
		}
	}
}
