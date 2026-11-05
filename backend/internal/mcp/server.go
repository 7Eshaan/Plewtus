package mcp

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"strings"
	"sync"

	"plewtus/internal/extractor"
	"plewtus/internal/model"
)

const ProtocolVersion = "2024-11-05"

// Searcher executes search requests.
type Searcher interface {
	Search(ctx context.Context, req *model.SearchRequest) (*model.SearchResponse, error)
}

// JSONRPCRequest represents an incoming JSON-RPC 2.0 message.
type JSONRPCRequest struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      any             `json:"id,omitempty"`
	Method  string          `json:"method"`
	Params  json.RawMessage `json:"params,omitempty"`
}

// JSONRPCResponse represents an outgoing JSON-RPC 2.0 message.
type JSONRPCResponse struct {
	JSONRPC string        `json:"jsonrpc"`
	ID      any           `json:"id,omitempty"`
	Result  any           `json:"result,omitempty"`
	Error   *JSONRPCError `json:"error,omitempty"`
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
	logger           *slog.Logger
	in               io.Reader
	out              io.Writer
	mu               sync.Mutex
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
		logger:           logger,
		in:               in,
		out:              out,
	}
}

// Serve reads JSON-RPC requests line-by-line and dispatches responses.
func (s *Server) Serve(ctx context.Context) error {
	scanner := bufio.NewScanner(s.in)
	// Allow larger buffer for requests carrying long arguments
	buf := make([]byte, 1024*1024)
	scanner.Buffer(buf, 10*1024*1024)

	for scanner.Scan() {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}

		s.handleLine(ctx, []byte(line))
	}

	return scanner.Err()
}

func (s *Server) handleLine(ctx context.Context, data []byte) {
	var req JSONRPCRequest
	if err := json.Unmarshal(data, &req); err != nil {
		s.writeError(nil, CodeParseError, "Parse error: "+err.Error())
		return
	}

	// Notifications (no ID)
	if req.ID == nil {
		s.handleNotification(&req)
		return
	}

	res, err := s.handleMethod(ctx, req.Method, req.Params)
	if err != nil {
		s.writeError(req.ID, err.Code, err.Message)
		return
	}

	s.writeResult(req.ID, res)
}

func (s *Server) handleNotification(req *JSONRPCRequest) {
	switch req.Method {
	case "notifications/initialized":
		s.logger.Debug("mcp client initialized notification received")
	default:
		s.logger.Debug("unhandled mcp notification", slog.String("method", req.Method))
	}
}

func (s *Server) handleMethod(ctx context.Context, method string, params json.RawMessage) (any, *JSONRPCError) {
	switch method {
	case "initialize":
		return map[string]any{
			"protocolVersion": ProtocolVersion,
			"capabilities": map[string]any{
				"tools": map[string]any{},
			},
			"serverInfo": map[string]any{
				"name":    "plewtus",
				"version": "0.1.0",
			},
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
	return []map[string]any{
		{
			"name":        "search",
			"description": "Search the web across multiple search engines (DuckDuckGo, Bing, Yahoo) with query-relevance filtering, hybrid relevance/consensus ranking, deduplication, URL unwrapping, and optional AI content extraction.",
			"inputSchema": map[string]any{
				"type": "object",
				"properties": map[string]any{
					"query": map[string]any{
						"type":        "string",
						"description": "Search query to execute across engines.",
					},
					"limit": map[string]any{
						"type":        "integer",
						"description": "Number of ranked results to return (default 10, max 50).",
					},
					"offset": map[string]any{
						"type":        "integer",
						"minimum":     0,
						"maximum":     49,
						"description": "Pagination offset (default 0); offset + limit must not exceed 50.",
					},
					"freshness": map[string]any{
						"type":        "string",
						"enum":        []string{"day", "week", "month", "year"},
						"description": "Optional recency filter.",
					},
					"include_domains": map[string]any{
						"type":        "array",
						"items":       map[string]any{"type": "string"},
						"description": "Only return results matching these hostnames/domains.",
					},
					"exclude_domains": map[string]any{
						"type":        "array",
						"items":       map[string]any{"type": "string"},
						"description": "Exclude results matching these hostnames/domains.",
					},
					"country": map[string]any{
						"type":        "string",
						"description": "2-letter ISO country code (e.g. 'us', 'gb').",
					},
					"language": map[string]any{
						"type":        "string",
						"description": "2-letter ISO language code (e.g. 'en', 'fr').",
					},
					"extract_mode": map[string]any{
						"type":        "string",
						"enum":        []string{"highlights", "markdown", "text", "metadata"},
						"description": "Optional webpage content extraction mode.",
					},
				},
				"required": []string{"query"},
			},
		},
		{
			"name":        "extract",
			"description": "Extract clean text, Markdown, or query highlights and metadata from one or more web URLs safely (with SSRF and size guards).",
			"inputSchema": map[string]any{
				"type": "object",
				"properties": map[string]any{
					"urls": map[string]any{
						"type":        "array",
						"maxItems":    10,
						"items":       map[string]any{"type": "string"},
						"description": "List of HTTP/HTTPS URLs to scrape and extract.",
					},
					"mode": map[string]any{
						"type":        "string",
						"enum":        []string{"markdown", "highlights", "text", "metadata"},
						"description": "Content extraction format (default 'markdown').",
					},
					"max_chars": map[string]any{
						"type":        "integer",
						"minimum":     1,
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

// searchToolArgs represents arguments for the 'search' tool.
type searchToolArgs struct {
	Query          string   `json:"query"`
	Limit          int      `json:"limit,omitempty"`
	Offset         int      `json:"offset,omitempty"`
	Freshness      string   `json:"freshness,omitempty"`
	IncludeDomains []string `json:"include_domains,omitempty"`
	ExcludeDomains []string `json:"exclude_domains,omitempty"`
	Country        string   `json:"country,omitempty"`
	Language       string   `json:"language,omitempty"`
	ExtractMode    string   `json:"extract_mode,omitempty"`
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
		return &ToolCallResult{
			Content: []ToolContent{{Type: "text", Text: "Error: query is required and cannot be empty"}},
			IsError: true,
		}, nil
	}

	req := &model.SearchRequest{
		Query:          args.Query,
		Limit:          args.Limit,
		Offset:         args.Offset,
		Freshness:      args.Freshness,
		IncludeDomains: args.IncludeDomains,
		ExcludeDomains: args.ExcludeDomains,
		Country:        args.Country,
		Language:       args.Language,
	}

	if args.ExtractMode != "" {
		req.Extract = &model.ExtractOptions{
			Enabled:  true,
			Mode:     args.ExtractMode,
			MaxPages: 3,
		}
	}

	req.Normalize(10, 50, s.defaultProviders)
	if err := req.Validate(); err != nil {
		return &ToolCallResult{
			Content: []ToolContent{{Type: "text", Text: "Validation Error: " + err.Error()}},
			IsError: true,
		}, nil
	}

	resp, err := s.searcher.Search(ctx, req)
	if err != nil {
		return &ToolCallResult{
			Content: []ToolContent{{Type: "text", Text: "Search Error: " + err.Error()}},
			IsError: true,
		}, nil
	}

	// Format response as clean JSON text for AI agent
	formattedBytes, err := json.MarshalIndent(resp, "", "  ")
	if err != nil {
		return nil, &JSONRPCError{Code: CodeInternalError, Message: err.Error()}
	}

	return &ToolCallResult{
		Content: []ToolContent{
			{
				Type: "text",
				Text: string(formattedBytes),
			},
		},
	}, nil
}

// extractToolArgs represents arguments for the 'extract' tool.
type extractToolArgs struct {
	URLs     []string `json:"urls"`
	Mode     string   `json:"mode,omitempty"`
	MaxChars int      `json:"max_chars,omitempty"`
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
		return &ToolCallResult{
			Content: []ToolContent{{Type: "text", Text: "Error: urls array is required and cannot be empty"}},
			IsError: true,
		}, nil
	}
	if len(args.URLs) > 10 {
		return &ToolCallResult{Content: []ToolContent{{Type: "text", Text: "Error: at most 10 URLs can be extracted per call"}}, IsError: true}, nil
	}

	mode := strings.ToLower(strings.TrimSpace(args.Mode))
	if mode == "" {
		mode = "markdown"
	}
	if mode != "markdown" && mode != "highlights" && mode != "text" && mode != "metadata" {
		return &ToolCallResult{Content: []ToolContent{{Type: "text", Text: "Error: mode must be markdown, highlights, text, or metadata"}}, IsError: true}, nil
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
	}

	var results []model.SearchResult
	for i, u := range args.URLs {
		results = append(results, model.SearchResult{
			ID:   fmt.Sprintf("ext_%d", i+1),
			URL:  u,
			Rank: i + 1,
		})
	}

	extracted := s.extractor.ExtractBatch(ctx, results, opts)

	formattedBytes, err := json.MarshalIndent(extracted, "", "  ")
	if err != nil {
		return nil, &JSONRPCError{Code: CodeInternalError, Message: err.Error()}
	}

	return &ToolCallResult{
		Content: []ToolContent{
			{
				Type: "text",
				Text: string(formattedBytes),
			},
		},
	}, nil
}

func (s *Server) writeResult(id any, result any) {
	s.mu.Lock()
	defer s.mu.Unlock()

	resp := JSONRPCResponse{
		JSONRPC: "2.0",
		ID:      id,
		Result:  result,
	}
	_ = json.NewEncoder(s.out).Encode(resp)
}

func (s *Server) writeError(id any, code int, msg string) {
	s.mu.Lock()
	defer s.mu.Unlock()

	resp := JSONRPCResponse{
		JSONRPC: "2.0",
		ID:      id,
		Error: &JSONRPCError{
			Code:    code,
			Message: msg,
		},
	}
	_ = json.NewEncoder(s.out).Encode(resp)
}
