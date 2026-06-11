package mcp

import (
	"context"
	"encoding/json"
	"strings"
	"testing"
)

func TestSafeToolsExposeOnlyReadOnlyPolygolemTools(t *testing.T) {
	tools := SafeTools()
	if len(tools) == 0 {
		t.Fatal("expected MCP tools")
	}
	for _, tool := range tools {
		if !strings.HasPrefix(tool.Name, "polygolem.") {
			t.Fatalf("tool name must be namespaced: %q", tool.Name)
		}
		lower := strings.ToLower(tool.Name + " " + tool.Description)
		for _, blocked := range []string{"trade", "sign", "approve", "withdraw", "cancel", "create_order"} {
			if strings.Contains(lower, blocked) {
				t.Fatalf("MCP v1 tool %q exposes blocked live/mutating wording %q", tool.Name, blocked)
			}
		}
		if tool.InputSchema["type"] != "object" {
			t.Fatalf("tool %s missing object input schema", tool.Name)
		}
		if tool.Name == "polygolem.marketdata_snapshot" {
			required, ok := tool.InputSchema["required"].([]string)
			if !ok || len(required) != 1 || required[0] != "token_id" {
				t.Fatalf("marketdata snapshot must require token_id: %#v", tool.InputSchema)
			}
		}
	}
}

func TestServerHandlesInitializeAndToolsList(t *testing.T) {
	server := NewServer()
	init := server.Handle(context.Background(), Request{JSONRPC: "2.0", ID: float64(1), Method: "initialize"})
	if init.Error != nil {
		t.Fatalf("initialize error: %+v", init.Error)
	}
	encoded, err := json.Marshal(init.Result)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(encoded), ProtocolVersion) {
		t.Fatalf("initialize missing protocol version: %s", encoded)
	}

	listed := server.Handle(context.Background(), Request{JSONRPC: "2.0", ID: "tools", Method: "tools/list"})
	if listed.Error != nil {
		t.Fatalf("tools/list error: %+v", listed.Error)
	}
	encoded, err = json.Marshal(listed.Result)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(encoded), "polygolem.health") {
		t.Fatalf("tools/list missing health tool: %s", encoded)
	}
}

func TestServerRejectsMutatingOrUnknownToolCalls(t *testing.T) {
	server := NewServer()
	params := json.RawMessage(`{"name":"polygolem.create_order"}`)
	got := server.Handle(context.Background(), Request{JSONRPC: "2.0", ID: 2, Method: "tools/call", Params: params})
	if got.Error == nil || got.Error.Code != -32602 {
		t.Fatalf("expected unknown mutating tool error, got %+v", got)
	}

	params = json.RawMessage(`{"name":"polygolem.health"}`)
	got = server.Handle(context.Background(), Request{JSONRPC: "2.0", ID: 3, Method: "tools/call", Params: params})
	if got.Error == nil || !strings.Contains(got.Error.Message, "not configured") {
		t.Fatalf("expected unconfigured execution error, got %+v", got)
	}
}

func TestServerExecutesConfiguredReadOnlyTool(t *testing.T) {
	server := NewServerWithHandlers(map[string]ToolHandler{
		"polygolem.health": func(ctx context.Context, arguments map[string]any) (any, error) {
			return map[string]string{"gamma": "ok", "clob": "ok"}, nil
		},
		"polygolem.create_order": func(ctx context.Context, arguments map[string]any) (any, error) {
			return "must be ignored", nil
		},
	})
	got := server.Handle(context.Background(), Request{
		JSONRPC: "2.0",
		ID:      4,
		Method:  "tools/call",
		Params:  json.RawMessage(`{"name":"polygolem.health","arguments":{}}`),
	})
	if got.Error != nil {
		t.Fatalf("tools/call error: %+v", got.Error)
	}
	encoded, err := json.Marshal(got.Result)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(encoded), `\"gamma\":\"ok\"`) || !strings.Contains(string(encoded), "content") {
		t.Fatalf("unexpected tool result: %s", encoded)
	}

	mutating := server.Handle(context.Background(), Request{JSONRPC: "2.0", ID: 5, Method: "tools/call", Params: json.RawMessage(`{"name":"polygolem.create_order"}`)})
	if mutating.Error == nil || mutating.Error.Code != -32602 {
		t.Fatalf("mutating handler should be ignored, got %+v", mutating)
	}
}
