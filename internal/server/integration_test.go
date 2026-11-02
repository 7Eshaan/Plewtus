package server

import (
	"bytes"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"plewtus/internal/config"
	"plewtus/internal/model"
	"plewtus/internal/orchestrator"
	"plewtus/internal/provider/all"
)

func TestSearchEndpoint_LiveIntegration(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping live end-to-end server integration test in short mode")
	}

	cfg := config.Load()
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	registry := all.NewRegistry(5*time.Second, cfg.UserAgent)
	orch := orchestrator.New(registry, 6*time.Second, cfg.UserAgent, logger)
	srv := New(cfg, logger, orch)

	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	payload := model.SearchRequest{
		Query:     "golang metasearch",
		Limit:     4,
		Providers: []string{"duckduckgo", "bing", "yahoo"},
	}
	data, err := json.Marshal(payload)
	if err != nil {
		t.Fatalf("failed to marshal payload: %v", err)
	}

	start := time.Now()
	resp, err := http.Post(ts.URL+"/search", "application/json", bytes.NewReader(data))
	elapsed := time.Since(start)

	if err != nil {
		t.Fatalf("POST /search failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		t.Fatalf("expected status 200, got %d: %s", resp.StatusCode, string(body))
	}

	var searchResp model.SearchResponse
	if err := json.NewDecoder(resp.Body).Decode(&searchResp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	t.Logf("Live search completed in %v. Results returned: %d. Successful: %v. Failed: %v",
		elapsed, len(searchResp.Results), searchResp.Metadata.Providers.Successful, searchResp.Metadata.Providers.Failed)

	if searchResp.Query != "golang metasearch" {
		t.Errorf("expected echoed query 'golang metasearch', got %q", searchResp.Query)
	}

	if len(searchResp.Metadata.Providers.Successful) == 0 {
		t.Fatalf("expected at least one successful provider, failed: %v", searchResp.Metadata.Providers.Failed)
	}

	if len(searchResp.Results) == 0 {
		t.Fatalf("expected search results from live query, got 0")
	}

	for i, r := range searchResp.Results {
		if r.ID == "" || r.Title == "" || r.URL == "" {
			t.Errorf("result %d missing ID/title/URL: %+v", i, r)
		}
		if r.Rank != i+1 {
			t.Errorf("result %d rank %d != %d", i, r.Rank, i+1)
		}
	}
}
