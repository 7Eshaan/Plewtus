package mcp

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"strings"
	"sync"
	"testing"
	"time"

	"plewtus/internal/model"
)

func runLines(t *testing.T, srv *Server, lines ...string) []JSONRPCResponse {
	t.Helper()
	out := &bytes.Buffer{}
	s := NewServer(srv.searcher, srv.extractor, strings.NewReader(strings.Join(lines, "\n")+"\n"), out, nil)
	if err := s.Serve(context.Background()); err != nil {
		t.Fatalf("Serve: %v", err)
	}
	var resps []JSONRPCResponse
	sc := bufio.NewScanner(out)
	for sc.Scan() {
		var r JSONRPCResponse
		if err := json.Unmarshal(sc.Bytes(), &r); err != nil {
			t.Fatalf("bad output line %q: %v", sc.Text(), err)
		}
		resps = append(resps, r)
	}
	return resps
}

func TestRegression_ParseErrorHasNullID(t *testing.T) {
	out := &bytes.Buffer{}
	s := NewServer(&mockSearcher{}, &mockExtractor{}, strings.NewReader("not json\n"), out, nil)
	_ = s.Serve(context.Background())
	if !strings.Contains(out.String(), `"id":null`) {
		t.Fatalf("parse error must carry id null, got %s", out.String())
	}
}

func TestRegression_ProtocolNegotiation(t *testing.T) {
	srv := NewServer(&mockSearcher{}, &mockExtractor{}, nil, nil, nil)
	resps := runLines(t, srv,
		`{"jsonrpc":"2.0","id":1,"method":"initialize","params":{"protocolVersion":"2025-06-18"}}`,
		`{"jsonrpc":"2.0","id":2,"method":"initialize","params":{"protocolVersion":"1999-01-01"}}`,
	)
	got := map[string]string{}
	for _, r := range resps {
		got[string(r.ID)] = r.Result.(map[string]any)["protocolVersion"].(string)
	}
	if got["1"] != "2025-06-18" || got["2"] != LatestProtocolVersion {
		t.Fatalf("unexpected negotiation: %v", got)
	}
}

type failingExtractor struct{ mockExtractor }

func (f *failingExtractor) Extract(ctx context.Context, u string, o *model.ExtractOptions) (*model.ExtractedContent, error) {
	if strings.Contains(u, "127.0.0.1") {
		return nil, errors.New("url validation failed: access to restricted IP address")
	}
	return &model.ExtractedContent{Markdown: "# Good page"}, nil
}

func TestRegression_ExtractReportsPerURLErrors(t *testing.T) {
	srv := NewServer(&mockSearcher{}, &failingExtractor{}, nil, nil, nil)
	resps := runLines(t, srv, `{"jsonrpc":"2.0","id":7,"method":"tools/call","params":{"name":"extract","arguments":{"urls":["https://ok.example","http://127.0.0.1/"]}}}`)
	res := resps[0].Result.(map[string]any)
	text := res["content"].([]any)[0].(map[string]any)["text"].(string)
	if !strings.Contains(text, "# Good page") || !strings.Contains(text, "restricted IP") {
		t.Fatalf("expected content and a per-URL error, got:\n%s", text)
	}
	if res["isError"] == true {
		t.Fatal("partial success must not be flagged as a tool error")
	}
}

func TestRegression_SearchTextFormatAndJSONOption(t *testing.T) {
	srv := NewServer(&mockSearcher{}, &mockExtractor{}, nil, nil, nil)
	resps := runLines(t, srv,
		`{"jsonrpc":"2.0","id":1,"method":"tools/call","params":{"name":"search","arguments":{"query":"go"}}}`,
		`{"jsonrpc":"2.0","id":2,"method":"tools/call","params":{"name":"search","arguments":{"query":"go","format":"json"}}}`,
	)
	for _, r := range resps {
		text := r.Result.(map[string]any)["content"].([]any)[0].(map[string]any)["text"].(string)
		switch string(r.ID) {
		case "1":
			if !strings.Contains(text, "1. Go Programming") || !strings.Contains(text, "https://go.dev") {
				t.Fatalf("unexpected text format:\n%s", text)
			}
		case "2":
			var parsed model.SearchResponse
			if err := json.Unmarshal([]byte(text), &parsed); err != nil || len(parsed.Results) != 1 {
				t.Fatalf("json format should be the API response: %v", err)
			}
		}
	}
}

type blockingSearcher struct{ started chan struct{} }

func (b *blockingSearcher) Search(ctx context.Context, req *model.SearchRequest) (*model.SearchResponse, error) {
	close(b.started)
	<-ctx.Done()
	return nil, ctx.Err()
}

// Regression: ping is answered while a tool call is running, and a
// notifications/cancelled message stops the call without a response.
func TestRegression_ConcurrentCallsAndCancellation(t *testing.T) {
	inR, inW := io.Pipe()
	out := &safeBuffer{}
	bs := &blockingSearcher{started: make(chan struct{})}
	s := NewServer(bs, &mockExtractor{}, inR, out, nil)
	done := make(chan error, 1)
	go func() { done <- s.Serve(context.Background()) }()

	_, _ = io.WriteString(inW, `{"jsonrpc":"2.0","id":"call-1","method":"tools/call","params":{"name":"search","arguments":{"query":"slow"}}}`+"\n")
	<-bs.started
	_, _ = io.WriteString(inW, `{"jsonrpc":"2.0","id":2,"method":"ping"}`+"\n")
	waitFor(t, func() bool { return strings.Contains(out.String(), `"id":2`) })
	_, _ = io.WriteString(inW, `{"jsonrpc":"2.0","method":"notifications/cancelled","params":{"requestId":"call-1"}}`+"\n")
	_ = inW.Close()

	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatal("Serve did not finish after cancellation and EOF")
	}
	if strings.Contains(out.String(), `"call-1"`) {
		t.Fatalf("cancelled call must not be answered, got %s", out.String())
	}
}

// Regression: shutdown signals stop the server even while stdin stays open.
func TestRegression_ServeStopsOnContextCancel(t *testing.T) {
	inR, inW := io.Pipe()
	defer inW.Close()
	s := NewServer(&mockSearcher{}, &mockExtractor{}, inR, &safeBuffer{}, nil)
	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan error, 1)
	go func() { done <- s.Serve(ctx) }()
	time.Sleep(50 * time.Millisecond)
	cancel()
	select {
	case err := <-done:
		if !errors.Is(err, context.Canceled) {
			t.Fatalf("expected context.Canceled, got %v", err)
		}
	case <-time.After(time.Second):
		t.Fatal("Serve kept running after its context was cancelled")
	}
}

func waitFor(t *testing.T, cond func() bool) {
	t.Helper()
	deadline := time.Now().Add(2 * time.Second)
	for !cond() {
		if time.Now().After(deadline) {
			t.Fatal("condition not met in time")
		}
		time.Sleep(10 * time.Millisecond)
	}
}

type safeBuffer struct {
	mu  sync.Mutex
	buf bytes.Buffer
}

func (b *safeBuffer) Write(p []byte) (int, error) {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.buf.Write(p)
}

func (b *safeBuffer) String() string {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.buf.String()
}

// Regression: a JSON-RPC batch is answered with one array of responses, with
// no entry for notifications.
func TestRegression_BatchRequests(t *testing.T) {
	out := &bytes.Buffer{}
	in := `[{"jsonrpc":"2.0","id":1,"method":"ping"},{"jsonrpc":"2.0","method":"notifications/initialized"},{"jsonrpc":"2.0","id":2,"method":"tools/call","params":{"name":"search","arguments":{"query":"go"}}},{"jsonrpc":"2.0","id":3,"method":"nope"}]` + "\n[]\n"
	s := NewServer(&mockSearcher{}, &mockExtractor{}, strings.NewReader(in), out, nil)
	if err := s.Serve(context.Background()); err != nil {
		t.Fatal(err)
	}
	lines := strings.Split(strings.TrimSpace(out.String()), "\n")
	if len(lines) != 2 {
		t.Fatalf("expected a batch response and an empty-batch error, got %d lines: %s", len(lines), out.String())
	}
	var batch []JSONRPCResponse
	var single JSONRPCResponse
	for _, line := range lines {
		if strings.HasPrefix(line, "[") {
			if err := json.Unmarshal([]byte(line), &batch); err != nil {
				t.Fatal(err)
			}
		} else if err := json.Unmarshal([]byte(line), &single); err != nil {
			t.Fatal(err)
		}
	}
	if len(batch) != 3 {
		t.Fatalf("expected 3 responses (none for the notification), got %d", len(batch))
	}
	byID := map[string]JSONRPCResponse{}
	for _, r := range batch {
		byID[string(r.ID)] = r
	}
	if byID["1"].Error != nil || byID["2"].Error != nil || byID["2"].Result == nil || byID["3"].Error == nil || byID["3"].Error.Code != CodeMethodNotFound {
		t.Fatalf("unexpected batch responses: %s", lines)
	}
	if single.Error == nil || single.Error.Code != CodeInvalidRequest {
		t.Fatalf("an empty batch must be an invalid request, got %+v", single)
	}
}
