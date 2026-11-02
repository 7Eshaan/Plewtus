package server

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"

	"plewtus/internal/config"
	"plewtus/internal/model"
)

type mockSearcher struct {
	response *model.SearchResponse
	err      error
}

func (m *mockSearcher) Search(ctx context.Context, req *model.SearchRequest) (*model.SearchResponse, error) {
	if m.err != nil {
		return nil, m.err
	}
	if m.response != nil {
		return m.response, nil
	}

	resp := model.NewSearchResponse(req.Query)
	resp.Results = []model.SearchResult{
		{
			ID:      "res_1",
			Title:   "Go Concurrency",
			URL:     "https://go.dev/concurrency",
			Snippet: "Concurrency in Go",
			Rank:    1,
			Score:   1.0,
			Engines: []model.EngineSource{{Name: "duckduckgo", Position: 1}},
		},
	}
	resp.Metadata.TotalResults = 1
	resp.Metadata.Providers.Queried = req.Providers
	resp.Metadata.Providers.Successful = req.Providers
	return resp, nil
}

func newTestServer(searcher ...Searcher) *Server {
	cfg := config.Load()
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	var s Searcher = &mockSearcher{}
	if len(searcher) > 0 && searcher[0] != nil {
		s = searcher[0]
	}
	return New(cfg, logger, s)
}

func TestHealthEndpoint(t *testing.T) {
	srv := newTestServer()
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	resp, err := http.Get(ts.URL + "/health")
	if err != nil {
		t.Fatalf("failed to GET /health: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("expected status 200, got %d", resp.StatusCode)
	}

	contentType := resp.Header.Get("Content-Type")
	if contentType != "application/json; charset=utf-8" {
		t.Errorf("expected json content-type, got %s", contentType)
	}

	var health model.HealthResponse
	if err := json.NewDecoder(resp.Body).Decode(&health); err != nil {
		t.Fatalf("failed to decode health response: %v", err)
	}

	if health.Status != "ok" {
		t.Errorf("expected status 'ok', got %q", health.Status)
	}
	if health.Version != "0.1.0" {
		t.Errorf("expected version '0.1.0', got %q", health.Version)
	}
}

func TestHealthzAliasEndpoint(t *testing.T) {
	srv := newTestServer()
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	resp, err := http.Get(ts.URL + "/healthz")
	if err != nil {
		t.Fatalf("failed to GET /healthz: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("expected status 200, got %d", resp.StatusCode)
	}
}

func TestSearchEndpoint_ValidRequest(t *testing.T) {
	srv := newTestServer()
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	reqPayload := model.SearchRequest{
		Query: "golang testing",
		Limit: 5,
	}
	data, _ := json.Marshal(reqPayload)

	resp, err := http.Post(ts.URL+"/search", "application/json", bytes.NewReader(data))
	if err != nil {
		t.Fatalf("failed to POST /search: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("expected status 200, got %d", resp.StatusCode)
	}

	var searchResp model.SearchResponse
	if err := json.NewDecoder(resp.Body).Decode(&searchResp); err != nil {
		t.Fatalf("failed to decode search response: %v", err)
	}

	if searchResp.Query != "golang testing" {
		t.Errorf("expected echoed query 'golang testing', got %q", searchResp.Query)
	}
	if searchResp.Results == nil {
		t.Errorf("expected non-nil results slice")
	}
	if len(searchResp.Results) != 1 {
		t.Errorf("expected 1 result, got %d", len(searchResp.Results))
	}
}

func TestSearchEndpoint_SearcherError(t *testing.T) {
	srv := newTestServer(&mockSearcher{err: errors.New("database outage")})
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	reqPayload := model.SearchRequest{
		Query: "golang testing",
	}
	data, _ := json.Marshal(reqPayload)

	resp, err := http.Post(ts.URL+"/search", "application/json", bytes.NewReader(data))
	if err != nil {
		t.Fatalf("failed to POST: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusInternalServerError {
		t.Errorf("expected status 500, got %d", resp.StatusCode)
	}

	var errResp model.ErrorResponse
	_ = json.NewDecoder(resp.Body).Decode(&errResp)
	if errResp.Error != "search_error" {
		t.Errorf("expected 'search_error', got %s", errResp.Error)
	}
}

func TestSearchEndpoint_SearcherTimeout(t *testing.T) {
	srv := newTestServer(&mockSearcher{err: context.DeadlineExceeded})
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	reqPayload := model.SearchRequest{
		Query: "golang testing",
	}
	data, _ := json.Marshal(reqPayload)

	resp, err := http.Post(ts.URL+"/search", "application/json", bytes.NewReader(data))
	if err != nil {
		t.Fatalf("failed to POST: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusRequestTimeout {
		t.Errorf("expected status 408, got %d", resp.StatusCode)
	}

	var errResp model.ErrorResponse
	_ = json.NewDecoder(resp.Body).Decode(&errResp)
	if errResp.Error != "request_timeout" {
		t.Errorf("expected 'request_timeout', got %s", errResp.Error)
	}
}

func TestSearchEndpoint_InvalidRequest(t *testing.T) {
	srv := newTestServer()
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	reqPayload := model.SearchRequest{
		Query: "",
	}
	data, _ := json.Marshal(reqPayload)

	resp, err := http.Post(ts.URL+"/search", "application/json", bytes.NewReader(data))
	if err != nil {
		t.Fatalf("failed to POST /search: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusBadRequest {
		t.Errorf("expected status 400 for empty query, got %d", resp.StatusCode)
	}
}

func TestSearchEndpoint_MalformedJSON(t *testing.T) {
	srv := newTestServer()
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	resp, err := http.Post(ts.URL+"/search", "application/json", bytes.NewReader([]byte("{invalid json")))
	if err != nil {
		t.Fatalf("failed to POST /search: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusBadRequest {
		t.Errorf("expected status 400 for malformed json, got %d", resp.StatusCode)
	}
}

func TestCORSOptions(t *testing.T) {
	srv := newTestServer()
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	req, _ := http.NewRequest(http.MethodOptions, ts.URL+"/search", nil)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("failed OPTIONS request: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusNoContent {
		t.Errorf("expected status 204 No Content for OPTIONS, got %d", resp.StatusCode)
	}
	if resp.Header.Get("Access-Control-Allow-Origin") != "*" {
		t.Errorf("expected CORS allow origin *, got %s", resp.Header.Get("Access-Control-Allow-Origin"))
	}
}
