package mcp

import (
	"context"
	"encoding/json"
	"strings"
	"testing"
	"time"
)

func TestNewReadOnlyHandlersExecutesWithTimeoutPolicy(t *testing.T) {
	var sawDeadline bool
	handlers := NewReadOnlyHandlers(HandlerConfig{
		Timeout: time.Second,
		Adapters: ReadOnlyAdapters{
			Health: func(ctx context.Context) (any, error) {
				_, sawDeadline = ctx.Deadline()
				return map[string]string{"gamma": "ok", "clob": "ok"}, nil
			},
		},
	})
	server := NewServerWithHandlers(handlers)
	got := server.Handle(context.Background(), Request{JSONRPC: "2.0", ID: 1, Method: "tools/call", Params: json.RawMessage(`{"name":"polygolem.health"}`)})
	if got.Error != nil {
		t.Fatalf("tools/call error: %+v", got.Error)
	}
	if !sawDeadline {
		t.Fatal("handler did not receive timeout-bound context")
	}
	encoded, _ := json.Marshal(got.Result)
	if !strings.Contains(string(encoded), `\"clob\":\"ok\"`) {
		t.Fatalf("unexpected result: %s", encoded)
	}
}

func TestNewReadOnlyHandlersValidatesRequiredArguments(t *testing.T) {
	handlers := NewReadOnlyHandlers(HandlerConfig{Adapters: ReadOnlyAdapters{
		DiscoverSearch: func(ctx context.Context, args map[string]any) (any, error) {
			return map[string]any{"query": args["query"]}, nil
		},
		MarketDataSnapshot: func(ctx context.Context, args map[string]any) (any, error) {
			return map[string]any{"token_id": args["token_id"]}, nil
		},
	}})
	server := NewServerWithHandlers(handlers)
	missing := server.Handle(context.Background(), Request{JSONRPC: "2.0", ID: 1, Method: "tools/call", Params: json.RawMessage(`{"name":"polygolem.discover_search","arguments":{}}`)})
	if missing.Error == nil || !strings.Contains(missing.Error.Message, "query is required") {
		t.Fatalf("expected missing query error, got %+v", missing)
	}

	ok := server.Handle(context.Background(), Request{JSONRPC: "2.0", ID: 2, Method: "tools/call", Params: json.RawMessage(`{"name":"polygolem.discover_search","arguments":{"query":"btc"}}`)})
	if ok.Error != nil {
		t.Fatalf("unexpected query error: %+v", ok.Error)
	}

	missingTokenID := server.Handle(context.Background(), Request{JSONRPC: "2.0", ID: 3, Method: "tools/call", Params: json.RawMessage(`{"name":"polygolem.marketdata_snapshot","arguments":{}}`)})
	if missingTokenID.Error == nil || !strings.Contains(missingTokenID.Error.Message, "token_id is required") {
		t.Fatalf("expected missing token_id error, got %+v", missingTokenID)
	}
}

func TestNewReadOnlyHandlersSkipsMissingAdapters(t *testing.T) {
	handlers := NewReadOnlyHandlers(HandlerConfig{})
	if len(handlers) != 0 {
		t.Fatalf("expected no handlers when adapters are missing: %#v", handlers)
	}
}
