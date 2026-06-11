package mcp

import (
	"context"
	"fmt"
	"time"
)

const defaultHandlerTimeout = 10 * time.Second

// ReadOnlyAdapters are the production read-only operations MCP may execute.
// Callers wire these to pkg clients or workflow runners; mutating operations are
// deliberately absent from this interface.
type ReadOnlyAdapters struct {
	Health             func(context.Context) (any, error)
	DiscoverSearch     func(context.Context, map[string]any) (any, error)
	DataPositions      func(context.Context, map[string]any) (any, error)
	OrderBook          func(context.Context, map[string]any) (any, error)
	MarketDataSnapshot func(context.Context, map[string]any) (any, error)
}

// HandlerConfig controls read-only MCP execution policy.
type HandlerConfig struct {
	Timeout  time.Duration
	Adapters ReadOnlyAdapters
}

// NewReadOnlyHandlers converts configured read-only adapters into MCP tool
// handlers. Each handler runs under a per-call timeout. Missing adapters are not
// registered, so tools remain discoverable but unconfigured calls fail safely.
func NewReadOnlyHandlers(config HandlerConfig) map[string]ToolHandler {
	timeout := config.Timeout
	if timeout <= 0 {
		timeout = defaultHandlerTimeout
	}
	handlers := map[string]ToolHandler{}
	add := func(name string, fn func(context.Context, map[string]any) (any, error)) {
		if fn == nil {
			return
		}
		handlers[name] = func(ctx context.Context, args map[string]any) (any, error) {
			callCtx, cancel := context.WithTimeout(ctx, timeout)
			defer cancel()
			return fn(callCtx, args)
		}
	}
	if config.Adapters.Health != nil {
		add("polygolem.health", func(ctx context.Context, args map[string]any) (any, error) {
			return config.Adapters.Health(ctx)
		})
	}
	add("polygolem.discover_search", requireStringArg("query", config.Adapters.DiscoverSearch))
	add("polygolem.data_positions", requireStringArg("user", config.Adapters.DataPositions))
	add("polygolem.orderbook_book", requireStringArg("token_id", config.Adapters.OrderBook))
	add("polygolem.marketdata_snapshot", requireStringArg("token_id", config.Adapters.MarketDataSnapshot))
	return handlers
}

func requireStringArg(name string, fn func(context.Context, map[string]any) (any, error)) func(context.Context, map[string]any) (any, error) {
	if fn == nil {
		return nil
	}
	return func(ctx context.Context, args map[string]any) (any, error) {
		value, ok := args[name].(string)
		if !ok || value == "" {
			return nil, fmt.Errorf("%s is required", name)
		}
		return fn(ctx, args)
	}
}
