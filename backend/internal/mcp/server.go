package mcp

import (
	"bufio"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"strings"
	"sync"

	"plewtus/internal/extractor"
	"plewtus/internal/model"
)

// LatestProtocolVersion is the newest MCP revision this server implements.
const LatestProtocolVersion = "2025-06-18"

// supportedProtocolVersions lists revisions this server can speak. The tool
// surface used here (initialize, ping, tools/list, tools/call) is identical
// across them.
var supportedProtocolVersions = map[string]bool{
	"2025-06-18": true,
	"2025-03-26": true,
	"2024-11-05": true,
}

// maxConcurrentCalls bounds how many tool calls run at once.
const maxConcurrentCalls = 4

// Searcher executes search requests.
type Searcher interface {
	Search(ctx context.Context, req *model.SearchRequest) (*model.SearchResponse, error)
}

// JSONRPCRequest represents an incoming JSON-RPC 2.0 message.
type JSONRPCRequest struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      json.RawMessage `json:"id,omitempty"`
	Method  string          `json:"method"`
	Params  json.RawMessage `json:"params,omitempty"`
}

// JSONRPCResponse represents an outgoing JSON-RPC 2.0 message.
type JSONRPCResponse struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      json.RawMessage `json:"id"`
	Result  any             `json:"result,omitempty"`
	Error   *JSONRPCError   `json:"error,omitempty"`
}

// JSONRPCError represents a JSON-RPC error object.
type JSONRPCError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
	Data    any    `json:"data,omitempty"`
}

// Standard JSON-RPC error codes.
const (
	CodeParseError     = -32700
	CodeInvalidRequest = -32600
	CodeMethodNotFound = -32601
	CodeInvalidParams  = -32602
	CodeInternalError  = -32603
)

// ToolContent represents a single content item returned by a tool call.
type ToolContent struct {
	Type string `json:"type"`
	Text string `json:"text"`
}

// ToolCallResult represents the result payload for tools/call.
type ToolCallResult struct {
	Content []ToolContent `json:"content"`
	IsError bool          `json:"isError,omitempty"`
}

// Server provides a Model Context Protocol (MCP) stdio server for AI agents.
type Server struct {
	searcher         Searcher
	extractor        extractor.ContentExtractor
	defaultProviders []string
	providerNames    []string
	defaultLimit     int
	maxLimit         int
	version          string
	logger           *slog.Logger
	in               io.Reader
	out              io.Writer

	writeMu sync.Mutex

	inflightMu sync.Mutex
	inflight   map[string]context.CancelFunc
}

// NewServer creates a new MCP Server instance.
func NewServer(searcher Searcher, ext extractor.ContentExtractor, in io.Reader, out io.Writer, logger *slog.Logger, defaultProviders ...[]string) *Server {
	if logger == nil {
		logger = slog.New(slog.NewTextHandler(io.Discard, nil))
	}
	providers := []string{"duckduckgo", "bing", "yahoo"}
	if len(defaultProviders) > 0 && len(defaultProviders[0]) > 0 {
		providers = append([]string(nil), defaultProviders[0]...)
	}
	return &Server{
		searcher:         searcher,
		extractor:        ext,
		defaultProviders: providers,
		providerNames:    []string{"duckduckgo", "bing", "yahoo"},
		defaultLimit:     10,
		maxLimit:         model.MaxResultWindow,
		version:          "0.1.0",
		logger:           logger,
		in:               in,
		out:              out,
		inflight:         make(map[string]context.CancelFunc),
	}
}

// WithVersion sets the server version reported during initialization.
func (s *Server) WithVersion(version string) *Server {
	if version != "" {
		s.version = version
	}
	return s
}

// WithLimits sets the default and maximum result counts for the search tool.
func (s *Server) WithLimits(defaultLimit, maxLimit int) *Server {
	if maxLimit > 0 {
		s.maxLimit = maxLimit
	}
	if defaultLimit > 0 {
		s.defaultLimit = defaultLimit
	}
	return s
}

// WithProviderNames sets the provider names advertised in the search tool schema.
func (s *Server) WithProviderNames(names []string) *Server {
	if len(names) > 0 {
		s.providerNames = append([]string(nil), names...)
	}
	return s
}

// Serve reads newline-delimited JSON-RPC messages and dispatches them. Tool
// calls run concurrently so that ping and cancellation stay responsive. Serve
// returns when the input ends (after in-flight calls finish) or immediately
// when ctx is cancelled.
func (s *Server) Serve(ctx context.Context) error {
	ctx, cancelAll := context.WithCancel(ctx)
	defer cancelAll()

	lines := make(chan []byte)
	scanErr := make(chan error, 1)
	go func() {
		defer close(lines)
		scanner := bufio.NewScanner(s.in)
		scanner.Buffer(make([]byte, 64*1024), 10*1024*1024)
		for scanner.Scan() {
			line := append([]byte(nil), scanner.Bytes()...)
			select {
			case lines <- line:
			case <-ctx.Done():
				return
			}
		}
		scanErr <- scanner.Err()
	}()

	var wg sync.WaitGroup
	sem := make(chan struct{}, maxConcurrentCalls)
	for {
		select {
		case <-ctx.Done():
			// Stop promptly on shutdown signals even while stdin stays open.
			return ctx.Err()
		case line, ok := <-lines:
			if !ok {
				wg.Wait()
				select {
				case err := <-scanErr:
					return err
				default:
					return nil
				}
			}
			s.dispatch(ctx, line, &wg, sem)
		}
	}
}

func (s *Server) dispatch(ctx context.Context, line []byte, wg *sync.WaitGroup, sem chan struct{}) {
	trimmed := strings.TrimSpace(string(line))
	if trimmed == "" {
		return
	}

	var req JSONRPCRequest
	if err := json.Unmarshal([]byte(trimmed), &req); err != nil {
		s.writeError(nil, CodeParseError, "Parse error: "+err.Error())
		return
	}
	if req.JSONRPC != "2.0" || req.Method == "" {
		if isNotification(req.ID) {
			return
		}
		s.writeError(req.ID, CodeInvalidRequest, "Invalid request: expected jsonrpc \"2.0\" and a method")
		return
	}

	if isNotification(req.ID) {
		s.handleNotification(&req)
		return
	}

	// Fast, synchronous methods are answered inline so their order is preserved.
	if req.Method != "tools/call" {
		res, rpcErr := s.handleMethod(ctx, req.Method, req.Params)
		s.reply(req.ID, res, rpcErr)
		return
	}

	callCtx, cancel := context.WithCancel(ctx)
	key := string(req.ID)
	s.inflightMu.Lock()
	s.inflight[key] = cancel
	s.inflightMu.Unlock()

	wg.Add(1)
	go func() {
		defer wg.Done()
		defer func() {
			cancel()
			s.inflightMu.Lock()
			delete(s.inflight, key)
			s.inflightMu.Unlock()
		}()
		select {
		case sem <- struct{}{}:
			defer func() { <-sem }()
		case <-callCtx.Done():
			return
		}
		defer func() {
			if rec := recover(); rec != nil {
				s.logger.Error("mcp tool call panic recovered", slog.Any("error", rec))
				s.writeError(req.ID, CodeInternalError, "internal error")
			}
		}()
		res, rpcErr := s.handleToolCall(callCtx, req.Params)
		if callCtx.Err() != nil && ctx.Err() == nil {
			// Cancelled by the client: the spec says not to respond.
			return
		}
		s.reply(req.ID, res, rpcErr)
	}()
}

func isNotification(id json.RawMessage) bool {
	return len(id) == 0
}

func (s *Server) reply(id json.RawMessage, res any, rpcErr *JSONRPCError) {
	if rpcErr != nil {
		s.writeError(id, rpcErr.Code, rpcErr.Message)
		return
	}
	s.writeResult(id, res)
}

func (s *Server) handleNotification(req *JSONRPCRequest) {
	switch req.Method {
	case "notifications/initialized":
		s.logger.Debug("mcp client initialized notification received")
	case "notifications/cancelled":
		var params struct {
			RequestID json.RawMessage `json:"requestId"`
		}
		if err := json.Unmarshal(req.Params, &params); err == nil && len(params.RequestID) > 0 {
			s.inflightMu.Lock()
			cancel := s.inflight[string(params.RequestID)]
			s.inflightMu.Unlock()
			if cancel != nil {
				cancel()
			}
		}
	default:
		s.logger.Debug("unhandled mcp notification", slog.String("method", req.Method))
	}
}

func (s *Server) handleMethod(ctx context.Context, method string, params json.RawMessage) (any, *JSONRPCError) {
	switch method {
	case "initialize":
		var init struct {
			ProtocolVersion string `json:"protocolVersion"`
		}
		_ = json.Unmarshal(params, &init)
		version := LatestProtocolVersion
		if supportedProtocolVersions[init.ProtocolVersion] {
			version = init.ProtocolVersion
		}
		return map[string]any{
			"protocolVersion": version,
			"capabilities": map[string]any{
				"tools": map[string]any{"listChanged": false},
			},
			"serverInfo": map[string]any{
				"name":    "plewtus",
				"title":   "Plewtus Metasearch",
				"version": s.version,
			},
			"instructions": "Use `search` to find web pages across DuckDuckGo, Bing, and Yahoo (optionally with page content), and `extract` to read specific URLs.",
		}, nil

	case "ping":
		return map[string]any{}, nil

	case "tools/list":
		return map[string]any{
			"tools": s.toolDefinitions(),
		}, nil

	case "tools/call":
		return s.handleToolCall(ctx, params)

	default:
		return nil, &JSONRPCError{
			Code:    CodeMethodNotFound,
			Message: fmt.Sprintf("Method %q not found", method),
		}
	}
}

func (s *Server) toolDefinitions() []map[string]any {
	readOnly := map[string]any{"readOnlyHint": true, "openWorldHint": true, "idempotentHint": true}
	return []map[string]any{
		{
			"name":        "search",
			"title":       "Web search",
			"description": "Search the web across DuckDuckGo, Bing, and Yahoo. Results are deduplicated, off-topic results are removed, and the rest are ranked by query relevance and cross-engine agreement. Optionally extracts page content from the top results.",
			"annotations": readOnly,
			"inputSchema": map[string]any{
				"type": "object",
				"properties": map[string]any{
					"query": map[string]any{
						"type":        "string",
						"minLength":   1,
						"description": "Search query to execute across engines.",
					},
					"limit": map[string]any{
						"type":        "integer",
						"minimum":     1,
						"maximum":     s.maxLimit,
						"description": fmt.Sprintf("Number of ranked results to return (default %d).", s.defaultLimit),
					},
					"offset": map[string]any{
						"type":        "integer",
						"minimum":     0,
						"maximum":     model.MaxResultWindow - 1,
						"description": fmt.Sprintf("Pagination offset (default 0); offset + limit must not exceed %d.", model.MaxResultWindow),
					},
					"providers": map[string]any{
						"type":        "array",
						"items":       map[string]any{"type": "string", "enum": s.providerNames},
						"description": "Engines to query (default: all).",
					},
					"freshness": map[string]any{
						"type":        "string",
						"enum":        []string{"day", "week", "month", "year"},
						"description": "Optional recency filter.",
					},
					"include_domains": map[string]any{
						"type":        "array",
						"items":       map[string]any{"type": "string"},
						"description": "Only return results from these domains (subdomains included).",
					},
					"exclude_domains": map[string]any{
						"type":        "array",
						"items":       map[string]any{"type": "string"},
						"description": "Exclude results from these domains.",
					},
					"country": map[string]any{
						"type":        "string",
						"pattern":     "^[A-Za-z]{2}$",
						"description": "2-letter ISO country code (e.g. 'us', 'gb').",
					},
					"language": map[string]any{
						"type":        "string",
						"pattern":     "^[A-Za-z]{2}$",
						"description": "2-letter ISO language code (e.g. 'en', 'fr').",
					},
					"extract_mode": map[string]any{
						"type":        "string",
						"enum":        []string{"highlights", "markdown", "text", "metadata"},
						"description": "Extract page content from the top results in this format.",
					},
					"max_pages": map[string]any{
						"type":        "integer",
						"minimum":     1,
						"maximum":     10,
						"description": "How many top results to extract when extract_mode is set (default 3).",
					},
					"max_chars_per_page": map[string]any{
						"type":        "integer",
						"minimum":     100,
						"maximum":     20000,
						"description": "Character limit per extracted page (default 4000).",
					},
					"format": map[string]any{
						"type":        "string",
						"enum":        []string{"text", "json"},
						"description": "Output format: compact readable text (default) or the full JSON API response.",
					},
				},
				"required": []string{"query"},
			},
		},
		{
			"name":        "extract",
			"title":       "Read web pages",
			"description": "Fetch one or more web pages and return clean Markdown, plain text, query highlights, or metadata. Private and local network addresses are blocked.",
			"annotations": readOnly,
			"inputSchema": map[string]any{
				"type": "object",
				"properties": map[string]any{
					"urls": map[string]any{
						"type":        "array",
						"minItems":    1,
						"maxItems":    10,
						"items":       map[string]any{"type": "string"},
						"description": "HTTP/HTTPS URLs to fetch.",
					},
					"mode": map[string]any{
						"type":        "string",
						"enum":        []string{"markdown", "highlights", "text", "metadata"},
						"description": "Content format (default 'markdown').",
					},
					"query": map[string]any{
						"type":        "string",
						"description": "Optional query used to pick the most relevant sentences in 'highlights' mode.",
					},
					"max_chars": map[string]any{
						"type":        "integer",
						"minimum":     100,
						"maximum":     20000,
						"description": "Maximum characters per page (default 4000).",
					},
				},
				"required": []string{"urls"},
			},
		},
	}
}

// ToolCallParams defines parameters passed to tools/call.
type ToolCallParams struct {
	Name      string          `json:"name"`
	Arguments json.RawMessage `json:"arguments"`
}

func (s *Server) handleToolCall(ctx context.Context, params json.RawMessage) (*ToolCallResult, *JSONRPCError) {
	var call ToolCallParams
	if err := json.Unmarshal(params, &call); err != nil {
		return nil, &JSONRPCError{
			Code:    CodeInvalidParams,
			Message: "Invalid tool call params: " + err.Error(),
		}
	}

	switch call.Name {
	case "search":
		return s.executeSearchTool(ctx, call.Arguments)
	case "extract":
		return s.executeExtractTool(ctx, call.Arguments)
	default:
		return nil, &JSONRPCError{
			Code:    CodeInvalidParams,
			Message: fmt.Sprintf("Unknown tool %q", call.Name),
		}
	}
}

func toolError(format string, args ...any) *ToolCallResult {
	return &ToolCallResult{Content: []ToolContent{{Type: "text", Text: fmt.Sprintf(format, args...)}}, IsError: true}
}

func toolText(text string) *ToolCallResult {
	return &ToolCallResult{Content: []ToolContent{{Type: "text", Text: text}}}
}

// searchToolArgs represents arguments for the 'search' tool.
type searchToolArgs struct {
	Query           string   `json:"query"`
	Limit           int      `json:"limit,omitempty"`
	Offset          int      `json:"offset,omitempty"`
	Providers       []string `json:"providers,omitempty"`
	Freshness       string   `json:"freshness,omitempty"`
	IncludeDomains  []string `json:"include_domains,omitempty"`
	ExcludeDomains  []string `json:"exclude_domains,omitempty"`
	Country         string   `json:"country,omitempty"`
	Language        string   `json:"language,omitempty"`
	ExtractMode     string   `json:"extract_mode,omitempty"`
	MaxPages        int      `json:"max_pages,omitempty"`
	MaxCharsPerPage int      `json:"max_chars_per_page,omitempty"`
	Format          string   `json:"format,omitempty"`
}

func (s *Server) executeSearchTool(ctx context.Context, argsRaw json.RawMessage) (*ToolCallResult, *JSONRPCError) {
	var args searchToolArgs
	if len(argsRaw) > 0 {
		if err := json.Unmarshal(argsRaw, &args); err != nil {
			return nil, &JSONRPCError{
				Code:    CodeInvalidParams,
				Message: "Invalid search arguments: " + err.Error(),
			}
		}
	}

	if strings.TrimSpace(args.Query) == "" {
		return toolError("Error: query is required and cannot be empty"), nil
	}
	format := strings.ToLower(strings.TrimSpace(args.Format))
	if format != "" && format != "text" && format != "json" {
		return toolError("Error: format must be 'text' or 'json'"), nil
	}

	req := &model.SearchRequest{
		Query:          args.Query,
		Limit:          args.Limit,
		Offset:         args.Offset,
		Providers:      args.Providers,
		Freshness:      args.Freshness,
		IncludeDomains: args.IncludeDomains,
		ExcludeDomains: args.ExcludeDomains,
		Country:        args.Country,
		Language:       args.Language,
	}

	if args.ExtractMode != "" {
		req.Extract = &model.ExtractOptions{
			Enabled:         true,
			Mode:            args.ExtractMode,
			MaxPages:        args.MaxPages,
			MaxCharsPerPage: args.MaxCharsPerPage,
		}
	}

	req.Normalize(s.defaultLimit, s.maxLimit, s.defaultProviders)
	if err := req.Validate(); err != nil {
		return toolError("Validation Error: %s", err.Error()), nil
	}

	resp, err := s.searcher.Search(ctx, req)
	if err != nil {
		if errors.Is(err, model.ErrInvalidRequest) {
			return toolError("Validation Error: %s", err.Error()), nil
		}
		return toolError("Search Error: %s", err.Error()), nil
	}

	if format == "json" {
		data, err := json.Marshal(resp)
		if err != nil {
			return nil, &JSONRPCError{Code: CodeInternalError, Message: err.Error()}
		}
		return toolText(string(data)), nil
	}
	return toolText(formatSearchResponse(resp)), nil
}

// extractToolArgs represents arguments for the 'extract' tool.
type extractToolArgs struct {
	URLs     []string `json:"urls"`
	Mode     string   `json:"mode,omitempty"`
	Query    string   `json:"query,omitempty"`
	MaxChars int      `json:"max_chars,omitempty"`
}

type extractOutcome struct {
	url     string
	content *model.ExtractedContent
	err     error
}

func (s *Server) executeExtractTool(ctx context.Context, argsRaw json.RawMessage) (*ToolCallResult, *JSONRPCError) {
	var args extractToolArgs
	if len(argsRaw) > 0 {
		if err := json.Unmarshal(argsRaw, &args); err != nil {
			return nil, &JSONRPCError{
				Code:    CodeInvalidParams,
				Message: "Invalid extract arguments: " + err.Error(),
			}
		}
	}

	if len(args.URLs) == 0 {
		return toolError("Error: urls array is required and cannot be empty"), nil
	}
	if len(args.URLs) > 10 {
		return toolError("Error: at most 10 URLs can be extracted per call"), nil
	}

	mode := strings.ToLower(strings.TrimSpace(args.Mode))
	if mode == "" {
		mode = "markdown"
	}
	if mode != "markdown" && mode != "highlights" && mode != "text" && mode != "metadata" {
		return toolError("Error: mode must be markdown, highlights, text, or metadata"), nil
	}
	maxChars := args.MaxChars
	if maxChars <= 0 {
		maxChars = 4000
	} else if maxChars > 20000 {
		maxChars = 20000
	}

	opts := &model.ExtractOptions{
		Enabled:         true,
		Mode:            mode,
		MaxPages:        len(args.URLs),
		MaxCharsPerPage: maxChars,
		Query:           strings.TrimSpace(args.Query),
	}

	outcomes := make([]extractOutcome, len(args.URLs))
	var wg sync.WaitGroup
	for i, u := range args.URLs {
		wg.Add(1)
		go func(i int, u string) {
			defer wg.Done()
			u = strings.TrimSpace(u)
			content, err := s.extractor.Extract(ctx, u, opts)
			outcomes[i] = extractOutcome{url: u, content: content, err: err}
		}(i, u)
	}
	wg.Wait()

	var b strings.Builder
	failures := 0
	for i, o := range outcomes {
		if i > 0 {
			b.WriteString("\n\n---\n\n")
		}
		fmt.Fprintf(&b, "## [%d] %s\n", i+1, o.url)
		if o.err != nil || o.content == nil {
			failures++
			msg := "no content extracted"
			if o.err != nil {
				msg = o.err.Error()
			}
			fmt.Fprintf(&b, "Error: %s\n", msg)
			continue
		}
		writeContent(&b, o.content, "")
	}

	result := toolText(strings.TrimSpace(b.String()))
	result.IsError = failures == len(outcomes)
	return result, nil
}

// formatSearchResponse renders a compact, model-friendly view of a search
// response: one numbered block per result plus a one-line execution summary.
func formatSearchResponse(resp *model.SearchResponse) string {
	var b strings.Builder
	meta := resp.Metadata
	fmt.Fprintf(&b, "Search results for %q: %d result(s) in %d ms", resp.Query, len(resp.Results), meta.TookMs)
	if len(meta.Providers.Successful) > 0 {
		fmt.Fprintf(&b, " · engines: %s", strings.Join(meta.Providers.Successful, ", "))
	}
	if len(meta.Providers.Failed) > 0 {
		var failed []string
		for _, f := range meta.Providers.Failed {
			failed = append(failed, fmt.Sprintf("%s (%s)", f.Engine, truncate(f.Error, 90)))
		}
		fmt.Fprintf(&b, " · unavailable: %s", strings.Join(failed, "; "))
	}
	if meta.Cached {
		b.WriteString(" · cached")
	}
	b.WriteString("\n")

	if len(resp.Results) == 0 {
		b.WriteString("\nNo results found. Try broader terms or remove filters.")
		return b.String()
	}

	for _, r := range resp.Results {
		fmt.Fprintf(&b, "\n%d. %s\n   %s\n", r.Rank, r.Title, r.URL)
		var sources []string
		for _, e := range r.Engines {
			sources = append(sources, fmt.Sprintf("%s#%d", e.Name, e.Position))
		}
		line := "   Found by: " + strings.Join(sources, ", ")
		if r.PublishedDate != "" {
			line += " · Published: " + r.PublishedDate
		}
		b.WriteString(line + "\n")
		if r.Snippet != "" {
			b.WriteString("   " + r.Snippet + "\n")
		}
		if r.Content != nil {
			writeContent(&b, r.Content, "   ")
		}
	}
	return strings.TrimRight(b.String(), "\n")
}

func writeContent(b *strings.Builder, c *model.ExtractedContent, indent string) {
	var metaParts []string
	if c.Title != "" && indent == "" {
		metaParts = append(metaParts, "Title: "+c.Title)
	}
	if c.Author != "" {
		metaParts = append(metaParts, "Author: "+c.Author)
	}
	if c.PublishedDate != "" && indent == "" { // search results already show their date
		metaParts = append(metaParts, "Published: "+c.PublishedDate)
	}
	if len(metaParts) > 0 {
		b.WriteString(indent + strings.Join(metaParts, " · ") + "\n")
	}
	switch {
	case c.Markdown != "":
		b.WriteString(indent + "Content (markdown):\n")
		b.WriteString(indentBlock(c.Markdown, indent+"   ") + "\n")
	case len(c.Highlights) > 0:
		b.WriteString(indent + "Highlights:\n")
		for _, h := range c.Highlights {
			b.WriteString(indent + "   - " + h + "\n")
		}
	case c.Text != "":
		b.WriteString(indent + "Content:\n")
		b.WriteString(indentBlock(c.Text, indent+"   ") + "\n")
	case c.Description != "":
		b.WriteString(indent + "Description: " + c.Description + "\n")
	}
}

func indentBlock(text, indent string) string {
	if indent == "" {
		return text
	}
	lines := strings.Split(text, "\n")
	for i, l := range lines {
		if l != "" {
			lines[i] = indent + l
		}
	}
	return strings.Join(lines, "\n")
}

func truncate(s string, n int) string {
	r := []rune(s)
	if len(r) <= n {
		return s
	}
	return string(r[:n-1]) + "…"
}

func (s *Server) writeResult(id json.RawMessage, result any) {
	s.write(JSONRPCResponse{JSONRPC: "2.0", ID: id, Result: result})
}

func (s *Server) writeError(id json.RawMessage, code int, msg string) {
	s.write(JSONRPCResponse{JSONRPC: "2.0", ID: id, Error: &JSONRPCError{Code: code, Message: msg}})
}

func (s *Server) write(resp JSONRPCResponse) {
	if len(resp.ID) == 0 {
		resp.ID = json.RawMessage("null") // JSON-RPC requires "id": null when unknown.
	}
	s.writeMu.Lock()
	defer s.writeMu.Unlock()
	if err := json.NewEncoder(s.out).Encode(resp); err != nil {
		s.logger.Error("failed writing mcp response", slog.Any("error", err))
	}
}
