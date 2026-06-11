// Package mcp exposes a minimal read-only Model Context Protocol surface for
// agent integrations.
//
// The v1 surface deliberately lists only no-credential, read-only Polygolem
// actions. Live trading, signing, approvals, withdrawals, and authenticated
// mutations are excluded.
package mcp

import (
	"context"
	"encoding/json"
	"fmt"
)

const ProtocolVersion = "2024-11-05"

// Tool is the MCP tool descriptor shape used by tools/list.
type Tool struct {
	Name        string         `json:"name"`
	Description string         `json:"description"`
	InputSchema map[string]any `json:"inputSchema"`
}

// Request is a JSON-RPC request accepted by Server.Handle.
type Request struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      any             `json:"id,omitempty"`
	Method  string          `json:"method"`
	Params  json.RawMessage `json:"params,omitempty"`
}

// Response is a JSON-RPC response emitted by Server.Handle.
type Response struct {
	JSONRPC string         `json:"jsonrpc"`
	ID      any            `json:"id,omitempty"`
	Result  any            `json:"result,omitempty"`
	Error   *ResponseError `json:"error,omitempty"`
}

// ResponseError is a JSON-RPC error.
type ResponseError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

// ToolHandler executes one read-only MCP tool. Arguments are the decoded
// JSON object from tools/call.params.arguments.
type ToolHandler func(ctx context.Context, arguments map[string]any) (any, error)

// Content is one MCP tool-result content block.
type Content struct {
	Type string `json:"type"`
	Text string `json:"text"`
}

// ToolResult is returned from a successful tools/call.
type ToolResult struct {
	Content []Content `json:"content"`
}

// Server owns the read-only MCP manifest and optional safe tool handlers.
type Server struct {
	tools    []Tool
	handlers map[string]ToolHandler
}

// NewServer returns a server exposing only safe read-only tools. It does not
// execute tools unless handlers are provided through NewServerWithHandlers.
func NewServer() *Server {
	return NewServerWithHandlers(nil)
}

// NewServerWithHandlers returns a server with caller-provided read-only tool
// handlers. Handlers for names outside SafeTools are ignored.
func NewServerWithHandlers(handlers map[string]ToolHandler) *Server {
	server := &Server{tools: SafeTools(), handlers: map[string]ToolHandler{}}
	for _, tool := range server.tools {
		if handler := handlers[tool.Name]; handler != nil {
			server.handlers[tool.Name] = handler
		}
	}
	return server
}

// SafeTools returns the read-only MCP v1 tool manifest.
func SafeTools() []Tool {
	return []Tool{
		{
			Name:        "polygolem.health",
			Description: "Check read-only Gamma and CLOB API reachability.",
			InputSchema: objectSchema(nil, nil),
		},
		{
			Name:        "polygolem.discover_search",
			Description: "Search Polymarket Gamma markets without credentials.",
			InputSchema: objectSchema([]string{"query"}, map[string]any{"query": stringSchema(), "limit": integerSchema()}),
		},
		{
			Name:        "polygolem.data_positions",
			Description: "Read public Data API positions for a user address.",
			InputSchema: objectSchema([]string{"user"}, map[string]any{"user": stringSchema(), "limit": integerSchema()}),
		},
		{
			Name:        "polygolem.orderbook_book",
			Description: "Read a public CLOB order book by token id.",
			InputSchema: objectSchema([]string{"token_id"}, map[string]any{"token_id": stringSchema()}),
		},
		{
			Name:        "polygolem.marketdata_snapshot",
			Description: "Return a normalized market-data snapshot for supplied stream events.",
			InputSchema: objectSchema([]string{"token_id"}, map[string]any{"token_id": stringSchema()}),
		},
	}
}

// Handle processes a single JSON-RPC MCP request.
func (s *Server) Handle(ctx context.Context, req Request) Response {
	if req.JSONRPC != "2.0" {
		return errorResponse(req.ID, -32600, "invalid JSON-RPC version")
	}
	switch req.Method {
	case "initialize":
		return Response{JSONRPC: "2.0", ID: req.ID, Result: map[string]any{
			"protocolVersion": ProtocolVersion,
			"serverInfo":      map[string]string{"name": "polygolem", "version": "dev"},
			"capabilities":    map[string]any{"tools": map[string]any{}},
		}}
	case "tools/list":
		return Response{JSONRPC: "2.0", ID: req.ID, Result: map[string]any{"tools": s.tools}}
	case "tools/call":
		var params struct {
			Name      string         `json:"name"`
			Arguments map[string]any `json:"arguments"`
		}
		_ = json.Unmarshal(req.Params, &params)
		if !s.isSafeTool(params.Name) {
			return errorResponse(req.ID, -32602, fmt.Sprintf("tool %q is not exposed by polygolem MCP", params.Name))
		}
		handler := s.handlers[params.Name]
		if handler == nil {
			return errorResponse(req.ID, -32000, "tool execution is not configured for this read-only MCP server")
		}
		result, err := handler(ctx, params.Arguments)
		if err != nil {
			return errorResponse(req.ID, -32000, err.Error())
		}
		text, err := marshalToolText(result)
		if err != nil {
			return errorResponse(req.ID, -32000, err.Error())
		}
		return Response{JSONRPC: "2.0", ID: req.ID, Result: ToolResult{Content: []Content{{Type: "text", Text: text}}}}
	default:
		return errorResponse(req.ID, -32601, "method not found")
	}
}

func (s *Server) isSafeTool(name string) bool {
	for _, tool := range s.tools {
		if tool.Name == name {
			return true
		}
	}
	return false
}

func errorResponse(id any, code int, message string) Response {
	return Response{JSONRPC: "2.0", ID: id, Error: &ResponseError{Code: code, Message: message}}
}

func marshalToolText(value any) (string, error) {
	if text, ok := value.(string); ok {
		return text, nil
	}
	raw, err := json.Marshal(value)
	if err != nil {
		return "", fmt.Errorf("marshal tool result: %w", err)
	}
	return string(raw), nil
}

func objectSchema(required []string, properties map[string]any) map[string]any {
	if properties == nil {
		properties = map[string]any{}
	}
	return map[string]any{"type": "object", "additionalProperties": false, "required": required, "properties": properties}
}

func stringSchema() map[string]any  { return map[string]any{"type": "string"} }
func integerSchema() map[string]any { return map[string]any{"type": "integer", "minimum": 1} }
