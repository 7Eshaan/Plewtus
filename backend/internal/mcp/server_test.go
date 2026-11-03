package mcp

import (
	"bytes"
	"context"
	"encoding/json"
	"strings"
	"testing"

	"plewtus/internal/model"
)

type mockSearcher struct {
	lastReq *model.SearchRequest
	resp    *model.SearchResponse
	err     error
}

func (m *mockSearcher) Search(ctx context.Context, req *model.SearchRequest) (*model.SearchResponse, error) {
	m.lastReq = req
	if m.err != nil {
		return nil, m.err
	}
	if m.resp != nil {
		return m.resp, nil
	}
	resp := model.NewSearchResponse(req.Query)
	resp.Results = []model.SearchResult{
		{
			ID:    "res_1",
			Title: "Go Programming",
			URL:   "https://go.dev",
			Rank:  1,
			Score: 0.016393,
		},
	}
	resp.Metadata.TotalResults = 1
	return resp, nil
}

type mockExtractor struct{}

func (m *mockExtractor) Extract(ctx context.Context, targetURL string, opts *model.ExtractOptions) (*model.ExtractedContent, error) {
	return &model.ExtractedContent{
		Markdown:    "# Extracted Header\nContent text.",
		ExtractedAt: "2026-11-03T10:00:00Z",
	}, nil
}

func (m *mockExtractor) ExtractBatch(ctx context.Context, results []model.SearchResult, opts *model.ExtractOptions) []model.SearchResult {
	for i := range results {
		results[i].Content = &model.ExtractedContent{
			Markdown:    "# Extracted Header\nContent text.",
			ExtractedAt: "2026-11-03T10:00:00Z",
		}
	}
	return results
}

func executeMCPRequest(t *testing.T, srv *Server, reqJSON string) *JSONRPCResponse {
	t.Helper()
	in := strings.NewReader(reqJSON + "\n")
	out := &bytes.Buffer{}

	testSrv := NewServer(srv.searcher, srv.extractor, in, out, nil)
	err := testSrv.Serve(context.Background())
	if err != nil {
		t.Fatalf("Serve failed: %v", err)
	}

	var resp JSONRPCResponse
	if err := json.Unmarshal(out.Bytes(), &resp); err != nil {
		t.Fatalf("Failed to parse response JSON: %v, raw: %s", err, out.String())
	}
	return &resp
}

func TestMCP_Initialize(t *testing.T) {
	srv := NewServer(&mockSearcher{}, &mockExtractor{}, nil, nil, nil)
	resp := executeMCPRequest(t, srv, `{"jsonrpc":"2.0","id":1,"method":"initialize","params":{"protocolVersion":"2024-11-05"}}`)

	if resp.ID != float64(1) {
		t.Fatalf("expected ID 1, got %v", resp.ID)
	}
	if resp.Error != nil {
		t.Fatalf("unexpected error: %+v", resp.Error)
	}

	resMap, ok := resp.Result.(map[string]any)
	if !ok {
		t.Fatalf("expected result map, got %T", resp.Result)
	}
	if resMap["protocolVersion"] != ProtocolVersion {
		t.Fatalf("expected protocolVersion %s, got %v", ProtocolVersion, resMap["protocolVersion"])
	}
	serverInfo := resMap["serverInfo"].(map[string]any)
	if serverInfo["name"] != "plewtus" {
		t.Fatalf("expected name 'plewtus', got %v", serverInfo["name"])
	}
}

func TestMCP_ToolsList(t *testing.T) {
	srv := NewServer(&mockSearcher{}, &mockExtractor{}, nil, nil, nil)
	resp := executeMCPRequest(t, srv, `{"jsonrpc":"2.0","id":2,"method":"tools/list"}`)

	if resp.Error != nil {
		t.Fatalf("unexpected error: %+v", resp.Error)
	}
	resMap := resp.Result.(map[string]any)
	tools := resMap["tools"].([]any)
	if len(tools) != 2 {
		t.Fatalf("expected 2 tools (search, extract), got %d", len(tools))
	}

	tool1 := tools[0].(map[string]any)
	if tool1["name"] != "search" {
		t.Fatalf("expected first tool 'search', got %v", tool1["name"])
	}

	tool2 := tools[1].(map[string]any)
	if tool2["name"] != "extract" {
		t.Fatalf("expected second tool 'extract', got %v", tool2["name"])
	}
}

func TestMCP_SearchToolCall(t *testing.T) {
	searcher := &mockSearcher{}
	srv := NewServer(searcher, &mockExtractor{}, nil, nil, nil)

	req := `{"jsonrpc":"2.0","id":3,"method":"tools/call","params":{"name":"search","arguments":{"query":"go concurrency","limit":5,"freshness":"week"}}}`
	resp := executeMCPRequest(t, srv, req)

	if resp.Error != nil {
		t.Fatalf("unexpected error: %+v", resp.Error)
	}

	resMap := resp.Result.(map[string]any)
	contents := resMap["content"].([]any)
	if len(contents) != 1 {
		t.Fatalf("expected 1 content item, got %d", len(contents))
	}

	text := contents[0].(map[string]any)["text"].(string)
	if !strings.Contains(text, "Go Programming") {
		t.Fatalf("expected search result in text content, got: %s", text)
	}

	if searcher.lastReq.Query != "go concurrency" {
		t.Fatalf("expected query 'go concurrency', got %s", searcher.lastReq.Query)
	}
	if searcher.lastReq.Limit != 5 {
		t.Fatalf("expected limit 5, got %d", searcher.lastReq.Limit)
	}
	if searcher.lastReq.Freshness != "week" {
		t.Fatalf("expected freshness 'week', got %s", searcher.lastReq.Freshness)
	}
}

func TestMCP_ExtractToolCall(t *testing.T) {
	srv := NewServer(&mockSearcher{}, &mockExtractor{}, nil, nil, nil)

	req := `{"jsonrpc":"2.0","id":4,"method":"tools/call","params":{"name":"extract","arguments":{"urls":["https://go.dev/doc"],"mode":"markdown"}}}`
	resp := executeMCPRequest(t, srv, req)

	if resp.Error != nil {
		t.Fatalf("unexpected error: %+v", resp.Error)
	}

	resMap := resp.Result.(map[string]any)
	contents := resMap["content"].([]any)
	if len(contents) != 1 {
		t.Fatalf("expected 1 content item, got %d", len(contents))
	}

	text := contents[0].(map[string]any)["text"].(string)
	if !strings.Contains(text, "Extracted Header") {
		t.Fatalf("expected extracted content in text, got: %s", text)
	}
}

func TestMCP_UnknownMethod(t *testing.T) {
	srv := NewServer(&mockSearcher{}, &mockExtractor{}, nil, nil, nil)
	resp := executeMCPRequest(t, srv, `{"jsonrpc":"2.0","id":5,"method":"unknown/method"}`)

	if resp.Error == nil {
		t.Fatal("expected error for unknown method")
	}
	if resp.Error.Code != CodeMethodNotFound {
		t.Fatalf("expected error code %d, got %d", CodeMethodNotFound, resp.Error.Code)
	}
}

func TestMCP_Notification(t *testing.T) {
	srv := NewServer(&mockSearcher{}, &mockExtractor{}, nil, nil, nil)
	in := strings.NewReader(`{"jsonrpc":"2.0","method":"notifications/initialized"}` + "\n")
	out := &bytes.Buffer{}

	testSrv := NewServer(srv.searcher, srv.extractor, in, out, nil)
	err := testSrv.Serve(context.Background())
	if err != nil {
		t.Fatalf("Serve failed: %v", err)
	}
	// Notifications must produce no response on stdout
	if out.Len() != 0 {
		t.Fatalf("expected no output for notification, got: %s", out.String())
	}
}
